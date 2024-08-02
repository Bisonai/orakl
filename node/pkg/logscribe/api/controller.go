package api

import (
	"encoding/json"

	"bisonai.com/orakl/node/pkg/db"
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

func insertLogs(c *fiber.Ctx) error {
	logEntries := new([]LogInsertModel)
	if err := c.BodyParser(logEntries); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		return err
	}

	bulkCopyEntries := [][]interface{}{}
	for _, entries := range *logEntries {
		bulkCopyEntries = append(bulkCopyEntries, []interface{}{entries.Service, entries.Timestamp, entries.Level, entries.Message, entries.Fields})
	}

	if len(bulkCopyEntries) > 0 {
		_, err := db.BulkCopy(c.Context(), "logs", []string{"service", "timestamp", "level", "message", "fields"}, bulkCopyEntries)
		if err != nil {
			log.Error().Err(err).Msg("Failed to bulk copy logs")
			return err
		}
	}

	return nil
}
