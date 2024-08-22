package api

import (
	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/logscribe/logprocessor"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type LogInsertModel = logprocessor.LogInsertModel

type Service struct {
	Service string `db:"service"`
}

func insertLogs(c *fiber.Ctx) error {
	defer func(c *fiber.Ctx) {
		logEntries := new([]LogInsertModel)
		if err := c.BodyParser(logEntries); err != nil {
			log.Error().Err(err).Msg("Failed to parse request body")
		}
		c.Locals("logsChannel").(chan *[]LogInsertModel) <- logEntries
		log.Debug().Msgf("%d logs inserted successfully", len(*logEntries))
	}(c)

	return c.Status(fiber.StatusOK).SendString("Request received successfully")
}

func processLogs(c *fiber.Ctx) error {
	defer func(c *fiber.Ctx) {
		services, err := db.QueryRows[Service](c.Context(), logprocessor.GetServicesQuery, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get services")
		}
		for _, service := range services {
			processedLogs := logprocessor.ProcessLogs(c.Context(), service.Service)
			if len(processedLogs) > 0 {
				c.Locals("logProcessor").(*logprocessor.LogProcessor).CreateGithubIssue(c.Context(), processedLogs, service.Service)
			}
		}
	}(c)
	return c.Status(fiber.StatusOK).SendString("Request received successfully")
}
