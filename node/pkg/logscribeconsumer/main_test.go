package logscribeconsumer

import (
	"context"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/logscribe/logprocessor"
)

type Count struct {
	Count int `db:"count"`
}

const (
	BulkLogsCopyInterval = 100 * time.Millisecond
	ProcessLogsInterval  = 100 * time.Millisecond
)

func startLogscribe(ctx context.Context, t *testing.T) {
	go func() {
		logProcessor, err := logprocessor.New(ctx, logprocessor.WithBulkLogsCopyInterval(BulkLogsCopyInterval))
		if err != nil {
			t.Errorf("failed to create logprocessor: %v", err)
		}
		err = logscribe.Run(ctx, logProcessor)
		if err != nil {
			t.Errorf("failed to start logscribe app: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)
}

func cleanup(ctx context.Context) {
	db.QueryWithoutResult(ctx, "DELETE FROM logs;", nil)
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	db.ClosePool()
	os.Exit(exitCode)
}
