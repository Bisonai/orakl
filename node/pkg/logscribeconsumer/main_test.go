package logscribeconsumer

import (
	"context"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe"
)

type Count struct {
	Count int `db:"count"`
}

func startLogscribe(ctx context.Context, t *testing.T) {
	go func() {
		err := logscribe.Run(ctx)
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
