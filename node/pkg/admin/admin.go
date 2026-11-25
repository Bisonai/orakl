package admin

import (
	"context"
	"fmt"
	"os"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/host"
	"bisonai.com/orakl/node/pkg/admin/metrics"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/utils/buildinfo"
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

	registerRoutes(app)

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

func registerRoutes(app *fiber.App) {
	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(buildinfo.Root("Orakl Node Admin API"))
	})
	v1.Get("/version", buildinfo.FiberHandler)

	fetcher.Routes(v1)
	aggregator.Routes(v1)
	config.Routes(v1)
	host.Routes(v1)
	metrics.Routes(v1)
}

func SyncOraklConfig(ctx context.Context) error {
	return config.InitSyncDb(ctx)
}
