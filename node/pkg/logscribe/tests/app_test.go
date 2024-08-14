package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/logscribe/logprocessor"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

const (
	BulkLogsCopyInterval   = 100 * time.Millisecond
	ProcessLogsIntervalSec = 1
	insertLogDataCount     = 10_000
)

func TestLogscribeRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cleanup(ctx)

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := logscribe.Run(ctx, nil)
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
		cron := cron.New()
		_, err := cron.AddFunc(fmt.Sprintf("@every %ds", ProcessLogsIntervalSec), func() { // Run once a week, midnight between Sat/Sun
			services, err := db.QueryRows[logprocessor.Service](ctx, logprocessor.GetServicesQuery, nil)
			if err != nil {
				t.Logf("error getting services: %v", err)
				return
			}
			for _, service := range services {
				logprocessor.ProcessLogs(ctx, service.Service)
			}
		})
		if err != nil {
			t.Logf("error adding cron job: %v", err)
		}

		logProcessor, err := logprocessor.New(
			ctx,
			logprocessor.WithCron(cron),
			logprocessor.WithBulkLogsCopyInterval(BulkLogsCopyInterval),
		)
		if err != nil {
			t.Logf("error creating logprocessor: %v", err)
		}

		err = logscribe.Run(
			ctx,
			logProcessor,
		)
		if err != nil {
			t.Logf("error running logscribe: %v", err)
		}
	}()

	logsData, err := getInsertLogData(insertLogDataCount)
	if err != nil {
		t.Fatalf("failed to get insert log data: %v", err)
	}
	response, err := request.RequestRaw(
		request.WithEndpoint("http://localhost:3000/api/v1/"),
		request.WithMethod("POST"),
		request.WithBody(logsData),
	)
	if err != nil {
		t.Fatalf("failed to insert logs: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("failed to insert logs: %v", err)
	}

	time.Sleep(2*BulkLogsCopyInterval + 2*ProcessLogsIntervalSec*time.Second)

	count, err := db.QueryRow[Count](ctx, "SELECT COUNT(*) FROM logs", nil)

	if err != nil {
		t.Fatalf("failed to count logs: %v", err)
	}

	assert.Equal(t, 0, count.Count)

	cleanup(ctx)
	cancel()
	wg.Wait()
}
