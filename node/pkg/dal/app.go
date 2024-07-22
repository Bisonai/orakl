package dal

import (
	"context"
	"errors"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/utils/initializer"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"

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

	controller, err := api.Setup(ctx, adminEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return err
	}

	app, err := initializer.Setup(ctx, controller, keyCache)
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
