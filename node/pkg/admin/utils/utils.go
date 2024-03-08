package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"

	"github.com/PuerkitoBio/goquery"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog/log"
)

type SetupInfo struct {
	Version string
	Bus     *bus.MessageBus
}

func Setup(setupInfo SetupInfo) (*fiber.App, error) {
	if setupInfo.Version == "" {
		setupInfo.Version = "test"
	}

	ctx := context.Background()
	_, err := db.GetPool(ctx)
	if err != nil {
		return nil, errors.New("error getting db pool")
	}

	_, err = db.GetRedisConn(ctx)
	if err != nil {
		return nil, errors.New("error getting redis conn")
	}

	app := fiber.New(fiber.Config{
		AppName:           "Node API " + setupInfo.Version,
		EnablePrintRoutes: true,
		ErrorHandler:      CustomErrorHandler,
	})

	app.Use(recover.New(
		recover.Config{
			EnableStackTrace:  true,
			StackTraceHandler: CustomStackTraceHandler,
		},
	))

	app.Use(cors.New())

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bus", setupInfo.Bus)
		return c.Next()
	})

	return app, nil

}

func CustomErrorHandler(c *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Set Content-Type: text/plain; charset=utf-8
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

	// Return status code with error message
	// | ${status} | ${ip} | ${method} | ${path} | ${error}",

	log.
		Info().
		Err(err).
		Int("status", code).
		Str("ip", c.IP()).
		Str("method", c.Method()).
		Str("path", c.Path()).
		Msg("error")

	return c.Status(code).SendString(err.Error())
}

func CustomStackTraceHandler(_ *fiber.Ctx, e interface{}) {
	stackTrace := strings.Split(string(debug.Stack()), "\n")
	var failPoint string

	for _, line := range stackTrace {
		if strings.Contains(line, "controller.go") {
			path := strings.Split(strings.TrimSpace(line), " ")[0]
			splitted := strings.Split(path, "/")
			failPoint = splitted[len(splitted)-2] + "/" + splitted[len(splitted)-1]

			break
		}
	}
	log.
		Info().
		Str("failPoint", failPoint).
		Msgf("panic: %v", e)

	_, _ = os.Stderr.WriteString(fmt.Sprintf("%s\n", debug.Stack())) //nolint:errcheck // This will never fail
}

func SendMessage(c *fiber.Ctx, to string, command string, args map[string]interface{}) (bus.Message, error) {
	var msg bus.Message

	messageBus, ok := c.Locals("bus").(*bus.MessageBus)
	if !ok {
		return msg, errors.New("bus is not found, failed to message fetcher")
	}

	msg = bus.Message{
		From: bus.ADMIN,
		To:   to,
		Content: bus.MessageContent{
			Command: command,
			Args:    args,
		},
		Response: make(chan bus.MessageResponse),
	}
	return msg, messageBus.Publish(msg)
}

func GetOraklConfigUrls(ctx context.Context) ([]string, error) {
	resp, err := http.Get("https://config.orakl.network/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var urls []string
	doc.Find("Body > div > table:nth-child(3) a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			urls = append(urls, "https://config.orakl.network/"+href)
		}
	})
	return urls, nil
}
