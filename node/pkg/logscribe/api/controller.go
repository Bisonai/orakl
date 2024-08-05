package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

func insertLogs(c *fiber.Ctx) error {
	defer func(c *fiber.Ctx) {
		logEntries := new([]LogInsertModel)
		if err := c.BodyParser(logEntries); err != nil {
			log.Error().Err(err).Msg("Failed to parse request body")
		}
		c.Locals("logsChannel").(chan *[]LogInsertModel) <- logEntries
	}(c)

	return c.Status(fiber.StatusOK).SendString("Request received successfully")
}
