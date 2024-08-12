package test

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/stretchr/testify/assert"
)

const (
	BulkLogsCopyInterval = 100 * time.Millisecond
	ProcessLogsInterval  = 100 * time.Millisecond
	insertLogDataCount   = 10_000
)

func TestLogscribeRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cleanup(ctx)

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		logscribe, err := logscribe.New(ctx)
		if err != nil {
			errChan <- err
			close(errChan)
			return
		}

		err = logscribe.Run(ctx)
		if err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		t.Fatalf("error running logscribe: %v", err)
	default:
	}

	response, err := request.RequestRaw(request.WithEndpoint("http://localhost:3000/api/v1/"))

	if response.StatusCode != http.StatusOK {
		t.Fatalf("error requesting logscribe: %v", err)
	}
	if response == nil {
		t.Fatalf("received nil response: %v", err)
	}

	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)

	}

	assert.Equal(t, string(resultBody), "Logscribe service")

	cancel()
	wg.Wait()
}

func TestProcessLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		logscribe, err := logscribe.New(
			ctx,
			logscribe.WithBulkLogsCopyInterval(BulkLogsCopyInterval),
		)
		if err != nil {
			t.Logf("error creating logscribe: %v", err)
		}

		err = logscribe.Run(ctx)
		if err != nil {
			t.Logf("error running logscribe: %v", err)
		}
	}()

	logsData, err := getInsertLogData(insertLogDataCount)
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

	time.Sleep(2*BulkLogsCopyInterval + 2*ProcessLogsInterval)

	count, err := db.QueryRow[Count](ctx, "SELECT COUNT(*) FROM logs", nil)

	if err != nil {
		t.Fatalf("failed to count logs: %v", err)
	}

	assert.Equal(t, 0, count.Count)

	cleanup(ctx)
	cancel()
	wg.Wait()
}
