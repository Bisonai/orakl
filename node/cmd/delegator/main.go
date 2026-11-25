package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"bisonai.com/orakl/node/pkg/delegator/metrics"
	"bisonai.com/orakl/node/pkg/delegator/sign"
	"bisonai.com/orakl/node/pkg/delegator/utils"
	"bisonai.com/orakl/node/pkg/utils/buildinfo"
	"bisonai.com/orakl/node/pkg/utils/loginit"

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
	buildinfo.Register("delegator")

	appConfig, err := utils.Setup("0.0.1")
	if err != nil {
		panic(err)
	}

	app := appConfig.App
	defer appConfig.Store.Close()
	SetSigningRouter(app.Group("/api/v1"))

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

// SetSigningRouter registers the delegator routes: no allowlist or
// transaction management API (use 'allow' on the CLI).
func SetSigningRouter(router fiber.Router) {
	router.Get("", func(c *fiber.Ctx) error {
		return c.SendString(buildinfo.Root("Orakl Network Delegator"))
	})
	router.Get("/version", buildinfo.FiberHandler)
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
	metrics.StartAllowlistPoller(ctx, appConfig.Store.Count, metrics.PollInterval)
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
