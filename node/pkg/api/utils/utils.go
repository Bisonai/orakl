package utils

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"bisonai.com/miko/node/pkg/api/secrets"
	"golang.org/x/crypto/scrypt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type AppConfig struct {
	Postgres *pgxpool.Pool
	App      *fiber.App
}

func IsTesting(c *fiber.Ctx) bool {
	testing, ok := c.Locals("testing").(bool)
	if !ok {
		// disable test mode if loading testing fails
		return false
	} else {
		return testing
	}
}

func GetPgx(c *fiber.Ctx) (*pgxpool.Pool, error) {
	con, ok := c.Locals("pgxConn").(*pgxpool.Pool)
	if !ok {
		return con, errors.New("failed to get pgxConn")
	} else {
		return con, nil
	}
}

func RawQueryWithoutReturn(c *fiber.Ctx, query string, args map[string]any) error {
	pgxPool, err := GetPgx(c)
	if err != nil {
		return err
	}

	rows, err := pgxPool.Query(c.Context(), query, pgx.NamedArgs(args))
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func QueryRow[T any](c *fiber.Ctx, query string, args map[string]any) (T, error) {
	var result T
	pgxPool, err := GetPgx(c)
	if err != nil {
		return result, err
	}

	rows, err := pgxPool.Query(c.Context(), query, pgx.NamedArgs(args))
	if err != nil {
		return result, err
	}

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	return result, err
}

func QueryRows[T any](c *fiber.Ctx, query string, args map[string]any) ([]T, error) {
	results := []T{}
	pgxPool, err := GetPgx(c)
	if err != nil {
		return results, err
	}

	rows, err := pgxPool.Query(c.Context(), query, pgx.NamedArgs(args))
	if err != nil {
		return results, err
	}

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) || (results == nil && err == nil) {
		return []T{}, nil
	}
	return results, err
}

func Setup(options ...string) (AppConfig, error) {
	var version string
	var appConfig AppConfig

	if len(options) > 0 {
		version = options[0]
	} else {
		version = "test"
	}

	config, err := LoadEnvVars()
	if err != nil {
		return appConfig, err
	}
	// pgsql connect
	pgxPool, pgxError := pgxpool.New(context.Background(), config["DATABASE_URL"].(string))
	if pgxError != nil {
		return appConfig, pgxError
	}

	testing, err := strconv.ParseBool(config["TEST_MODE"].(string))
	if err != nil {
		// defaults to testing false
		testing = false
	}

	app := fiber.New(fiber.Config{
		AppName:           "Miko API " + version,
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
		c.Locals("pgxConn", pgxPool)
		c.Locals("testing", testing)
		return c.Next()
	})

	appConfig = AppConfig{
		Postgres: pgxPool,
		App:      app,
	}
	return appConfig, nil
}

func EncryptText(textToEncrypt string) (string, error) {
	config, err := LoadEnvVars()
	if err != nil {
		return "", err
	}
	password := config["ENCRYPT_PASSWORD"].(string)
	// Generate a random 16-byte IV
	iv := make([]byte, 16)
	if _, err = rand.Read(iv); err != nil {
		return "", err
	}

	// Derive a 32-byte key using scrypt
	key, err := scrypt.Key([]byte(password), []byte("salt"), 16384, 8, 1, 32)
	if err != nil {
		return "", err
	}

	// Create a cipher using AES-256-CTR with the key and IV
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	stream := cipher.NewCTR(block, iv)

	// Encrypt the text
	ciphertext := make([]byte, len(textToEncrypt))
	stream.XORKeyStream(ciphertext, []byte(textToEncrypt))

	// Combine the IV and ciphertext into a single string
	encryptedText := hex.EncodeToString(iv) + hex.EncodeToString(ciphertext)

	return encryptedText, nil
}

func DecryptText(encryptedText string) (string, error) {
	config, err := LoadEnvVars()
	if err != nil {
		return "", err
	}
	password := config["ENCRYPT_PASSWORD"].(string)

	// Extract the IV and ciphertext from the string
	iv, err := hex.DecodeString(encryptedText[:32])
	if err != nil {
		return "", err
	}
	ciphertext, err := hex.DecodeString(encryptedText[32:])
	if err != nil {
		return "", err
	}

	// Derive the key using scrypt
	key, err := scrypt.Key([]byte(password), []byte("salt"), 16384, 8, 1, 32)
	if err != nil {
		return "", err
	}

	// Create a decipher using AES-256-CTR with the key and IV
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	stream := cipher.NewCTR(block, iv)

	// Decrypt the ciphertext
	decryptedText := make([]byte, len(ciphertext))
	stream.XORKeyStream(decryptedText, ciphertext)

	return string(decryptedText), nil
}

func LoadEnvVars() (map[string]interface{}, error) {
	databaseURL := secrets.GetSecret("DATABASE_URL")
	encryptPassword := secrets.GetSecret("ENCRYPT_PASSWORD")

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	appPort := os.Getenv("APP_PORT")
	testMode := os.Getenv("TEST_MODE")

	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	if redisHost == "" {
		redisHost = "localhost"
	}
	if redisPort == "" {
		redisPort = "6379"
	}
	if appPort == "" {
		appPort = "3000"
	}
	if encryptPassword == "" {
		return nil, errors.New("ENCRYPT_PASSWORD is not set")
	}

	return map[string]interface{}{
		"DATABASE_URL":     databaseURL,
		"REDIS_HOST":       redisHost,
		"REDIS_PORT":       redisPort,
		"APP_PORT":         appPort,
		"TEST_MODE":        testMode,
		"ENCRYPT_PASSWORD": encryptPassword,
	}, nil
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
	log.Error().Err(err).Str("call info", fmt.Sprintf("| %d | %s | %s | %s | %s\n", code, c.IP(), c.Method(), c.Path(), err.Error()))
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
	log.Debug().Any("stacktrace", stackTrace).Str("failPoint", failPoint).Msgf("panic: %v", e)
	_, _ = os.Stderr.WriteString(fmt.Sprintf("%s\n", debug.Stack())) //nolint:errcheck // This will never fail
}
