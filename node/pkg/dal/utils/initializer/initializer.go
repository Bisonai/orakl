package initializer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/utils/stats"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var keyCache *keycache.KeyCache

func Setup(ctx context.Context) (*fiber.App, error) {
	keyCache = keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	_, err := db.GetPool(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting pgs conn in Setup")
		return nil, errorSentinel.ErrAdminDbPoolNotFound
	}

	_, err = db.GetRedisClient(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting redis conn in Setup")
		return nil, errorSentinel.ErrAdminRedisConnNotFound
	}

	app := fiber.New(fiber.Config{
		AppName:           "Data Availability Layer API 0.1.0",
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
	app.Use(keyauth.New(keyauth.Config{
		Next:      authFilter,
		KeyLookup: "header:X-API-Key",
		Validator: validator,
	}))

	app.Use(stats.StatsMiddleware)
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
		Error().
		Err(err).
		Int("status", code).
		Str("ip", c.IP()).
		Str("method", c.Method()).
		Str("path", c.Path()).
		Msg("error")

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
	return c.Status(code).JSON(fiber.Map{"error": err.Error()})
}

func CustomStackTraceHandler(_ *fiber.Ctx, e interface{}) {
	stackTrace := strings.Split(string(debug.Stack()), "\n")
	var failPoint string

	for _, line := range stackTrace {
		if strings.Contains(line, ".go") {
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

func authFilter(c *fiber.Ctx) bool {
	originalURL := strings.ToLower(c.OriginalURL())
	return originalURL == "/api/v1"
}

func validator(c *fiber.Ctx, s string) (bool, error) {
	if s == "" {
		return false, fmt.Errorf("missing api key")
	}

	if keyCache.Get(s) {
		return true, nil
	}

	if validateApiKeyFromDB(c.Context(), s) {
		keyCache.Set(s)
		return true, nil
	}

	return false, fmt.Errorf("invalid api key")
}

func validateApiKeyFromDB(ctx context.Context, apiKey string) bool {
	err := db.QueryWithoutResult(ctx, "SELECT 1 FROM keys WHERE key = @key", map[string]any{"key": apiKey})
	if err != nil && err == pgx.ErrNoRows {
		log.Error().Err(err).Msg("error validating api key")
		return false
	}
	return true
}
