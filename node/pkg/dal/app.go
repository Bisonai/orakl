package dal

import (
	"context"
	"errors"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/collector"
	"bisonai.com/orakl/node/pkg/dal/utils/initializer"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	adminEndpoint := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if adminEndpoint == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL is not set")
	}

	configs, err := fetchConfigs(ctx, adminEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch configs")
		return err
	}

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup collector")
		return err
	}

	hub := api.HubSetup(ctx, configs)

	app, err := initializer.Setup(ctx, collector, hub, keyCache)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return err
	}
	defer func() {
		_ = app.Shutdown()
	}()

	v1 := app.Group("")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Node DAL API")
	})
	api.Routes(v1)

	port := os.Getenv("DAL_API_PORT")
	if port == "" {
		port = "8090"
	}

	return app.Listen(":" + port)
}

func fetchConfigs(ctx context.Context, endpoint string) ([]types.Config, error) {
	return request.Request[[]types.Config](request.WithEndpoint(endpoint + "/config"))
}
