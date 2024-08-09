package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/logscribe/api"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog/log"
)

const batchSize = 1000

func Setup(appVersion string, logsChannel chan *[]api.LogInsertModel) (*fiber.App, error) {
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

func hashLog(log api.LogInsertModel) string {
	hash := sha256.New()
	hash.Write([]byte(log.Service))
	hash.Write([]byte(log.Timestamp.Format(time.RFC3339)))
	hash.Write([]byte(fmt.Sprintf("%d", log.Level)))
	hash.Write([]byte(log.Message))
	hash.Write(log.Fields)
	return hex.EncodeToString(hash.Sum(nil))
}

func FetchAndProcessLogs(ctx context.Context) (*[]api.LogInsertModel, error) {
	processedLogs := make([]api.LogInsertModel, 0)
	logMap := make(map[string]bool, 0)
	offset := 0

	for {
		logs, err := db.QueryRows[api.LogInsertModel](ctx, api.ReadLogs, map[string]any{
			"limit":  batchSize,
			"offset": offset,
		})
		if err != nil {
			return nil, err
		}
		if len(logs) == 0 {
			break
		}

		for _, log := range logs {
			hash := hashLog(log)
			if !logMap[hash] {
				processedLogs = append(processedLogs, log)
				logMap[hash] = true
			}
		}

		offset += batchSize
	}

	return &processedLogs, nil
}
