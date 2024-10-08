package logscribe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"bisonai.com/miko/node/pkg/db"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/logscribe/logprocessor"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog/log"
)

func Setup(appVersion string, logsChannel chan *[]LogInsertModel, p *logprocessor.LogProcessor) (*fiber.App, error) {
	ctx := context.Background()
	_, err := db.GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting db pool")
		return nil, errorSentinel.ErrLogscribeDbPoolNotFound
	}

	app := fiber.New(fiber.Config{
		AppName:           "Logscribe " + appVersion,
		EnablePrintRoutes: true,
		ErrorHandler:      CustomErrorHandler,
	})

	app.Use(recover.New(
		recover.Config{
			EnableStackTrace:  true,
			StackTraceHandler: CustomStackTraceHandler,
		},
	))

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("logsChannel", logsChannel)
		c.Locals("logProcessor", p)
		return c.Next()
	})

	app.Use(cors.New())

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
