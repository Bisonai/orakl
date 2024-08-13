package logscribe

import (
	"context"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"bisonai.com/orakl/node/pkg/logscribe/logprocessor"
	"github.com/gofiber/fiber/v2"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

const (
	logsChannelSize             = 10_000
	DefaultBulkLogsCopyInterval = 30 * time.Second
)

func New(ctx context.Context, options ...AppOption) (*App, error) {
	c := &AppConfig{
		bulkLogsCopyInterval: DefaultBulkLogsCopyInterval,
	}
	for _, option := range options {
		option(c)
	}

	logProcessor, err := logprocessor.New(ctx)
	if err != nil {
		return nil, err
	}

	return &App{
		bulkLogsCopyInterval: c.bulkLogsCopyInterval,
		cron:                 c.cron,
		logProcessor:         logProcessor,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Debug().Msg("Starting logscribe server")

	logsChannel := make(chan *[]LogInsertModel, logsChannelSize)

	fiberApp, err := Setup("0.1.0", logsChannel, a.logProcessor)
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

	go a.bulkCopyLogs(ctx, logsChannel)

	if err := a.startProcessingCronJob(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	if err := fiberApp.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown logscribe server")
		return err
	}

	return nil
}

func (a *App) startProcessingCronJob(ctx context.Context) error {
	if a.cron == nil {
		cron := cron.New(cron.WithSeconds())
		_, err := cron.AddFunc("0 0 0 * * 5", func() { // Run once a week, midnight between Thu/Fri
			services, err := db.QueryRows[Service](ctx, logprocessor.GetServicesQuery, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get services")
				return
			}
			for _, service := range services {
				processedLogs := logprocessor.ProcessLogs(ctx, service.Service)
				if len(processedLogs) > 0 {
					a.logProcessor.CreateGithubIssue(ctx, processedLogs, service.Service)
				}
			}
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to start processing cron job")
			return err
		}
		a.cron = cron
	}
	a.cron.Start()
	return nil
}

func (a *App) bulkCopyLogs(ctx context.Context, logsChannel <-chan *[]LogInsertModel) {
	ticker := time.NewTicker(a.bulkLogsCopyInterval)
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
					for _, log := range *logs {
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
}
