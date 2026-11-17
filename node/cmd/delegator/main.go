package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"bisonai.com/miko/node/pkg/delegator/contract"
	"bisonai.com/miko/node/pkg/delegator/function"
	"bisonai.com/miko/node/pkg/delegator/metrics"
	"bisonai.com/miko/node/pkg/delegator/organization"
	"bisonai.com/miko/node/pkg/delegator/reporter"
	"bisonai.com/miko/node/pkg/delegator/sign"
	"bisonai.com/miko/node/pkg/delegator/utils"
	"bisonai.com/miko/node/pkg/utils/loginit"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/kaiachain/kaia/common"
	"github.com/rs/zerolog/log"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Info().Msg("env file is not found, continuing without .env file")
	}

	if len(os.Args) > 1 && os.Args[1] == "token" {
		if err := runToken(context.Background(), os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "allow" {
		if err := runAllow(context.Background(), os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	loginit.InitZeroLog()

	appConfig, err := utils.Setup("0.0.1")
	if err != nil {
		panic(err)
	}

	app := appConfig.App
	v1 := app.Group("/api/v1")
	if appConfig.Store != nil {
		defer appConfig.Store.Close()
		SetSigningRouter(v1)
	} else {
		defer appConfig.Postgres.Close()
		SetRouter(v1)
	}

	if err := startMetrics(context.Background(), appConfig); err != nil {
		panic(err)
	}

	var port string
	port = os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	err = app.Listen(":" + port)
	if err != nil {
		panic(err)
	}

}

// SetSigningRouter registers the SQLite-mode routes: no allowlist or
// transaction management API (use 'allow' on the CLI).
func SetSigningRouter(router fiber.Router) {
	router.Get("", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Network Delegator")
	})
	sign.SigningRoutes(router)
}

// startMetrics serves metrics on METRICS_ADDR (empty: disabled) and starts
// the fee payer balance (needs PROVIDER_URL) and allowlist size pollers.
func startMetrics(ctx context.Context, appConfig utils.AppConfig) error {
	srv, err := metrics.Start(os.Getenv(metrics.EnvAddr))
	if err != nil || srv == nil {
		return err
	}
	metrics.StartBalancePoller(ctx, os.Getenv("PROVIDER_URL"), feePayerAddress, metrics.PollInterval)
	metrics.StartAllowlistPoller(ctx, allowlistCounter(appConfig), metrics.PollInterval)
	return nil
}

// feePayerAddress returns the address of the loaded fee payer key, if any.
func feePayerAddress() (common.Address, bool) {
	pk := utils.CurrentFeePayer()
	if pk == "" {
		return common.Address{}, false
	}
	addr, err := utils.GetPublicKey(pk)
	if err != nil {
		return common.Address{}, false
	}
	return common.HexToAddress(addr), true
}

func allowlistCounter(appConfig utils.AppConfig) func(context.Context) (int, error) {
	if appConfig.Store != nil {
		return appConfig.Store.Count
	}
	// functions.contract_id is a NOT NULL foreign key, and an existing
	// contract row is its approval, so every function row is signable
	return func(ctx context.Context) (int, error) {
		var n int
		err := appConfig.Postgres.QueryRow(ctx, "SELECT COUNT(*) FROM functions").Scan(&n)
		return n, err
	}
}

func SetRouter(router fiber.Router) {
	router.Get("", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Network Delegator")
	})

	contract.Routes(router)
	sign.Routes(router)
	function.Routes(router)
	organization.Routes(router)
	reporter.Routes(router)

}
