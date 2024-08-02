package logscribe

import (
	"context"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"bisonai.com/orakl/node/pkg/logscribe/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

const (
	logsChannelSize             = 10_000
	DefaultBulkLogsCopyInterval = 3 * time.Second
)

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting logscribe server")

	app, err := utils.Setup("0.1.0")
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup logscribe server")
		return err
	}

	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Logscribe service")
	})

	logsChannel := make(chan []api.LogInsertModel, logsChannelSize)

	port := os.Getenv("LOGSCRIBE_PORT")
	if port == "" {
		port = "3000"
	}
	api.Routes(v1, logsChannel)

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatal().Err(err).Msg("Failed to start logscribe server")
		}
	}()

	go func() {
		ticker := time.NewTicker(DefaultBulkLogsCopyInterval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				bulkCopyEntries := [][]interface{}{}
			loop:
				for {
					select {
					case logs := <-logsChannel:
						for _, log := range logs {
							bulkCopyEntries = append(bulkCopyEntries, []interface{}{log.Service, log.Timestamp, log.Level, log.Message, log.Fields})
						}
					default:
						break loop
					}
				}

				if len(bulkCopyEntries) > 0 {
					_, err := db.BulkCopy(ctx, "logs", []string{"service", "timestamp", "level", "message", "fields"}, bulkCopyEntries)
					if err != nil {
						log.Error().Err(err).Msg("Failed to bulk copy logs")
					}
				}
			}
		}
	}()

	<-ctx.Done()

	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown logscribe server")
		return err
	}

	return nil
}
