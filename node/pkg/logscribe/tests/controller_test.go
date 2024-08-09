package test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/stretchr/testify/assert"
)

type Count struct {
	Count int `db:"count"`
}

const insertLogDataCountController = 1000

func TestInsertLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logscribe, err := logscribe.New(ctx, logscribe.WithBulkLogsCopyInterval(BulkLogsCopyInterval))
		if err != nil {
			t.Logf("error creating logscribe: %v", err)
		}

		err = logscribe.Run(ctx)
		if err != nil {
			t.Logf("error running logscribe: %v", err)
		}
	}()

	logsData, err := getInsertLogData(insertLogDataCountController)
	if err != nil {
		t.Fatalf("failed to get insert log data: %v", err)
	}
	response, err := request.RequestRaw(request.WithEndpoint("http://localhost:3000/api/v1/"), request.WithMethod("POST"), request.WithBody(logsData))
	if err != nil {
		t.Fatalf("failed to insert logs: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("failed to insert logs: %v", err)
	}

	time.Sleep(2 * BulkLogsCopyInterval)

	count, err := db.QueryRow[Count](ctx, "SELECT COUNT(*) FROM logs", nil)

	if err != nil {
		t.Fatalf("failed to count logs: %v", err)
	}

	assert.Equal(t, insertLogDataCountController, count.Count)

	cleanup(ctx)
	cancel()
	wg.Wait()
}
