package admin

import (
	"context"
	"fmt"
	"os"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/providerUrl"
	"bisonai.com/orakl/node/pkg/admin/proxy"
	"bisonai.com/orakl/node/pkg/admin/reporter"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/admin/wallet"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Run(bus *bus.MessageBus) error {
	log.Debug().Msg("Starting admin server")
	app, err := utils.Setup(utils.SetupInfo{
		Version: "0.1.0",
		Bus:     bus,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup admin server")
		return err
	}

	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Node Admin API")
	})

	feed.Routes(v1)
	proxy.Routes(v1)
	fetcher.Routes(v1)
	aggregator.Routes(v1)
	reporter.Routes(v1)
	wallet.Routes(v1)
	providerUrl.Routes(v1)
	config.Routes(v1)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8088"
	}

	err = app.Listen(fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start admin server")
		return err
	}

	return nil
}

func SyncOraklConfig(ctx context.Context) error {
	return config.InitSyncDb(ctx)
}
