package utils

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"bisonai.com/orakl/node/pkg/delegator/secrets"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klaytn/klaytn/crypto"
)

type FeePayer struct {
	PrivateKey string `json:"privateKey" db:"privateKey"`
}

type AppConfig struct {
	Postgres *pgxpool.Pool
	App      *fiber.App
}

var feePayer string

func Setup(options ...string) (AppConfig, error) {
	var version string
	var appConfig AppConfig

	if len(options) > 0 {
		version = options[0]
	} else {
		version = "test"
	}

	pgxPool, pgxError := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if pgxError != nil {
		return appConfig, pgxError
	}

	err := InitFeePayerPK(context.Background(), pgxPool)
	if err != nil {
		fmt.Println("fee payer not initialized due to error:" + err.Error() + "\nplease refresh the application after fee payer insertion through following endpoint: /api/v1/sign/initialize")
	}

	app := fiber.New(fiber.Config{
		AppName:           "delegator " + version,
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
		c.Locals("feePayer", feePayer)
		c.Locals("pgxConn", pgxPool)
		c.Locals("validContracts", new(sync.Map))
		return c.Next()
	})

	appConfig = AppConfig{
		Postgres: pgxPool,
		App:      app,
	}
	return appConfig, nil
}

func InitFeePayerPK(ctx context.Context, pgxPool *pgxpool.Pool) error {
	var err error
	if feePayer = os.Getenv("DELEGATOR_FEEPAYER_PK"); feePayer != "" {
		return nil
	}

	useGoogleSecretManager, _ := strconv.ParseBool(os.Getenv("USE_GOOGLE_SECRET_MANAGER"))
	useVault, _ := strconv.ParseBool(os.Getenv("USE_VAULT"))
	if useVault {
		feePayer, err = LoadFeePayerFromVault(ctx)
		if err != nil {
			return err
		}
	} else if useGoogleSecretManager {
		feePayer, err = LoadFeePayerFromGSM(ctx)
		if err != nil {
			return err
		}
	}
	return nil
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
	log.Printf("| %d | %s | %s | %s | %s\n", code, c.IP(), c.Method(), c.Path(), err.Error())
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
	log.Printf("| (%s) panic: %v \n", failPoint, e)
	_, _ = os.Stderr.WriteString(fmt.Sprintf("%s\n", debug.Stack())) //nolint:errcheck // This will never fail
}

func GetFeePayer(c *fiber.Ctx) (string, error) {
	payer, ok := c.Locals("feePayer").(string)
	if !ok {
		return payer, errors.New("failed to get feePayer")
	} else {
		return payer, nil
	}
}

func UpdateFeePayer(newFeePayer string) {
	feePayer = newFeePayer
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
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}
	return results, err
}

func QueryRowWithoutFiberCtx[T any](postgres *pgxpool.Pool, query string, args map[string]any) (T, error) {
	var result T

	rows, err := postgres.Query(context.Background(), query, pgx.NamedArgs(args))
	if err != nil {
		return result, err
	}

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}

	return result, err
}

func QueryRowsWithoutFiberCtx[T any](postgres *pgxpool.Pool, query string, args map[string]any) ([]T, error) {
	results := []T{}

	rows, err := postgres.Query(context.Background(), query, pgx.NamedArgs(args))
	if err != nil {
		return results, err
	}

	results, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return results, nil
	}
	return results, err
}

func LoadFeePayerFromGSM(ctx context.Context) (string, error) {
	/*
		When you're running your application on Google Kubernetes Engine (GKE),
		you typically don't need to manually set the GOOGLE_APPLICATION_CREDENTIALS environment variable.

		By default, applications running on GKE have access to the Compute Engine default service account.
		This default service account has broad access, so it's recommended to limit its access by creating and using a custom service account.

		To use a custom service account, you can:

		- Create a new service account in the Google Cloud Console.
		- Grant the necessary IAM roles to the service account.
		- Create a new GKE node pool with the service account.
		- (Optional) If you want to use the service account for a specific workload instead of the whole node, you can use the Workload Identity feature.

		Once you've set up the service account, the Google Cloud client libraries can automatically use its credentials,
		so you don't need to manually set GOOGLE_APPLICATION_CREDENTIALS.
		The client libraries can also automatically refresh the credentials when they expire.
	*/
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: os.Getenv("GOOGLE_SECRET_PATH"),
	}

	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		panic(fmt.Errorf("failed to access secret version: %v", err))
	}

	if string(result.Payload.Data) == "" {
		return "", fmt.Errorf("failed to get secret from secret manager")
	}

	feePayer := result.Payload.Data
	pk := strings.TrimPrefix(string(feePayer), "0x")
	return pk, nil
}

func GetPublicKey(pk string) (string, error) {
	pk = strings.TrimPrefix(pk, "0x")

	if len(pk) == 110 {
		return "", fmt.Errorf("klaytn wallet key is given instead of private key")
	}

	privateKeyECDSA, err := crypto.HexToECDSA(pk)
	if err != nil {
		return "", fmt.Errorf("failed to convert private key to ECDSA: " + err.Error())
	}

	publicKey := privateKeyECDSA.Public()

	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to convert public key to ECDSA format: " + err.Error())
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address.String(), nil
}

func LoadFeePayerFromVault(ctx context.Context) (string, error) {
	feePayer := ""
	vaultRole := os.Getenv("VAULT_ROLE")
	jwtPath := os.Getenv("JWT_PATH")
	vaultSecretPath := os.Getenv("VAULT_SECRET_PATH")
	vaultKeyName := os.Getenv("VAULT_KEY_NAME")

	if vaultRole != "" && jwtPath != "" && vaultSecretPath != "" && vaultKeyName != "" {
		secretsEnv := secrets.SecretEnv{
			VaultRole:       vaultRole,
			JwtPath:         jwtPath,
			VaultSecretPath: vaultSecretPath,
			VaultKeyName:    vaultKeyName,
		}

		secrets, err := secretsEnv.GetSecretFromVaultWithKubernetesAuth()
		if err != nil {
			return "", err
		}
		feePayer = secrets.FeePayer
	}
	feePayer = strings.TrimPrefix(string(feePayer), "0x")
	return feePayer, nil
}
