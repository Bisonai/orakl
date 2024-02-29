package admin

import (
	"fmt"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/proxy"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func Run(port string, bus *bus.MessageBus) error {
	log.Debug().Msg("Starting admin server")
	app, err := utils.Setup(utils.SetupInfo{
		Version: "test",
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
	adapter.Routes(v1)
	feed.Routes(v1)
	proxy.Routes(v1)
	fetcher.Routes(v1)

	err = app.Listen(fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start admin server")
		return err
	}
	return nil
}
