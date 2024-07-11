package dal

import (
	"context"
	"errors"
	"os"

	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")
	app, err := utils.Setup(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return err
	}
	defer func() {
		_ = app.Shutdown()
	}()

	adminEndpoint := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if adminEndpoint == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL is not set")
	}

	err = api.Setup(ctx, adminEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return err
	}
	api.ApiController.Start(ctx)

	log.Debug().Str("Player", "DAL API").Msg("DAL API collector started")
	v1 := app.Group("/api/v1")
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
