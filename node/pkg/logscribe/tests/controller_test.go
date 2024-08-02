package test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/utils/request"
)

func TestInsertLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cleanup(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := logscribe.Run(ctx)
		if err != nil {
			t.Logf("error running logscribe: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

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

	cancel()
	wg.Wait()
}
