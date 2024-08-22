package test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/logscribe"
	"bisonai.com/miko/node/pkg/logscribe/logprocessor"
	"bisonai.com/miko/node/pkg/utils/request"
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
		logProcessor, err := logprocessor.New(ctx, logprocessor.WithBulkLogsCopyInterval(BulkLogsCopyInterval))
		if err != nil {
			t.Logf("failed to create log processor: %v", err)
		}
		err = logscribe.Run(ctx, logProcessor)
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
