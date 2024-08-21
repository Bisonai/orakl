package logscribe

import (
	"context"
	"os"

	"bisonai.com/miko/node/pkg/logscribe/api"
	"bisonai.com/miko/node/pkg/logscribe/logprocessor"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

const logsChannelSize = 10_000

type LogInsertModel = api.LogInsertModel

func Run(ctx context.Context, logProcessor *logprocessor.LogProcessor) error {
	log.Debug().Msg("Starting logscribe server")

	logsChannel := make(chan *[]LogInsertModel, logsChannelSize)

	var err error
	if logProcessor == nil {
		logProcessor, err = logprocessor.New(ctx)
		if err != nil {
			return err
		}
	}

	go logProcessor.BulkCopyLogs(ctx, logsChannel)

	if err = logProcessor.StartProcessingCronJob(ctx); err != nil {
		return err
	}

	fiberApp, err := Setup("0.1.0", logsChannel, logProcessor)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup logscribe server")
		return err
	}

	v1 := fiberApp.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Logscribe service")
	})

	port := os.Getenv("LOGSCRIBE_PORT")
	if port == "" {
		port = "3000"
	}
	api.Routes(v1)

	go func() {
		if err := fiberApp.Listen(":" + port); err != nil {
			log.Fatal().Err(err).Msg("Failed to start logscribe server")
		}
	}()

	<-ctx.Done()

	if err := fiberApp.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown logscribe server")
		return err
	}

	return nil
}
