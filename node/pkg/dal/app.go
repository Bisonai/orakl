package dal

import (
	"context"
	"os"

	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/utils/initializer"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")
	app, err := initializer.Setup(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return err
	}

	err = api.Setup(ctx)
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
