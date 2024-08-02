package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type LogInsertModel struct {
	Service   string          `db:"service" json:"service"`
	Timestamp string          `db:"timestamp" json:"timestamp"`
	Level     int             `db:"level" json:"level"`
	Message   string          `db:"message" json:"message"`
	Fields    json.RawMessage `db:"fields" json:"fields"`
}

func insertLogs(c *fiber.Ctx, logsChannel chan []LogInsertModel) error {
	logEntries := new([]LogInsertModel)
	if err := c.BodyParser(logEntries); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return err
	}
	logsChannel <- *logEntries

	return nil
}
