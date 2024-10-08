package admin

import (
	"context"
	"fmt"
	"os"

	"bisonai.com/miko/node/pkg/admin/aggregator"
	"bisonai.com/miko/node/pkg/admin/config"
	"bisonai.com/miko/node/pkg/admin/feed"
	"bisonai.com/miko/node/pkg/admin/fetcher"
	"bisonai.com/miko/node/pkg/admin/host"
	"bisonai.com/miko/node/pkg/admin/providerUrl"
	"bisonai.com/miko/node/pkg/admin/proxy"

	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Run(ctx context.Context, bus *bus.MessageBus) error {
	log.Debug().Msg("Starting admin server")
	app, err := utils.Setup(ctx, utils.SetupInfo{
		Version: "0.1.0",
		Bus:     bus,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup admin server")
		return err
	}

	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Miko Node Admin API")
	})

	feed.Routes(v1)
	proxy.Routes(v1)
	fetcher.Routes(v1)
	aggregator.Routes(v1)
	providerUrl.Routes(v1)
	config.Routes(v1)
	host.Routes(v1)

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

func SyncMikoConfig(ctx context.Context) error {
	return config.InitSyncDb(ctx)
}
