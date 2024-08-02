package test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/sentinel/pkg/db"
	"github.com/stretchr/testify/assert"
)

type Count struct {
	Count int `db:"count"`
}

func TestInsertLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := logscribe.Run(ctx)
		if err != nil {
			t.Logf("error running logscribe: %v", err)
		}
	}()

	logsData, err := getInsertLogData()
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

	time.Sleep(logscribe.DefaultBulkLogsCopyInterval + 1*time.Second)

	count, err := db.QueryRow[Count](ctx, "SELECT COUNT(*) FROM logs", nil)

	if err != nil {
		t.Fatalf("failed to count logs: %v", err)
	}

	assert.Equal(t, insertLogDataCount, count.Count)

	cleanup(ctx)
	cancel()
	wg.Wait()
}
