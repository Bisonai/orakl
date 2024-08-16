//nolint:all
package logscribeconsumer

import (
	"context"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"bisonai.com/orakl/node/pkg/logscribe/logprocessor"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type Count struct {
	Count int `db:"count"`
}

const (
	BulkLogsCopyInterval = 100 * time.Millisecond
	ProcessLogsInterval  = 100 * time.Millisecond
	logsChannelSize      = 10_000
)

func startLogscribe(ctx context.Context, t *testing.T) func() {
	logsChannel := make(chan *[]logprocessor.LogInsertModel, logsChannelSize)
	logProcessor, err := logprocessor.New(ctx, logprocessor.WithBulkLogsCopyInterval(BulkLogsCopyInterval))
	if err != nil {
		t.Fatalf("Failed to create log processor: %v", err)
		return nil
	}
	go logProcessor.BulkCopyLogs(ctx, logsChannel)

	if err = logProcessor.StartProcessingCronJob(ctx); err != nil {
		t.Fatalf("Failed to start processing cron job: %v", err)
		return nil
	}

	fiberApp, err := logscribe.Setup("0.1.0", logsChannel, logProcessor)
	if err != nil {
		t.Fatalf("Failed to setup logscribe server: %v", err)
		return nil
	}

	v1 := fiberApp.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Logscribe service")
	})
	api.Routes(v1)

	go func() {
		if err := fiberApp.Listen(":3000"); err != nil {
			log.Fatal().Err(err).Msg("Failed to start logscribe server")
		}
	}()

	return func() {
		cleanup(ctx, fiberApp)
	}
}

func cleanup(ctx context.Context, fiberApp *fiber.App) {
	db.QueryWithoutResult(ctx, "DELETE FROM logs;", nil)
	fiberApp.Shutdown()
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	db.ClosePool()
	os.Exit(exitCode)
}
