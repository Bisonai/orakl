package api

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type LogInsertModel struct {
	Service   string          `db:"service" json:"service"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
	Level     int             `db:"level" json:"level"`
	Message   string          `db:"message" json:"message"`
	Fields    json.RawMessage `db:"fields" json:"fields"`
}

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
