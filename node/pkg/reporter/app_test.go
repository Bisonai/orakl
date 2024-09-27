// //nolint:all
package reporter

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/db"
)

func TestFetchConfigs(t *testing.T) {
	configs, err := fetchConfigs()
	if err != nil || configs == nil || len(configs) == 0 {
		t.Fatalf("error getting configs: %v", err)
	}
}

func TestRunApp(t *testing.T) {
	ctx := context.Background()

	app := New()
	err := app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}
}

func TestWsDataHandling(t *testing.T) {
	ctx := context.Background()

	app := New()

	conn, tmpConfig, symbols, err := mockDalWsServer(ctx)
	if err != nil {
		t.Fatalf("error mocking dal ws server: %v", err)
	}

	app.WsHelper = conn
	go app.WsHelper.Run(ctx, app.HandleWsMessage)
	time.Sleep(100 * time.Millisecond)

	err = conn.Write(ctx, hub.Subscription{
		Method: "SUBSCRIBE",
		Params: []string{"submission@test-aggregate"},
	})
	if err != nil {
		t.Fatalf("error subscribing to websocket: %v", err)
	}

	sampleSubmissionData, err := generateSampleSubmissionData(
		tmpConfig.ID,
		int64(15),
		time.Now(),
		1,
		"test-aggregate",
	)
	if err != nil {
		t.Fatalf("error generating sample submission data: %v", err)
	}

	err = db.Publish(ctx, keys.SubmissionDataStreamKey("test-aggregate"), sampleSubmissionData)
	if err != nil {
		t.Fatalf("error publishing sample submission data: %v", err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.After(10 * time.Second)
	submissionDataCount := 0

	for {
		select {
		case <-ticker.C:
			if app.WsHelper != nil && app.WsHelper.IsRunning {
				submissionDataCount = 0
				for _, symbol := range symbols {
					if _, ok := app.LatestDataMap.Load(symbol); ok {
						submissionDataCount++
					}
				}
				if submissionDataCount == len(symbols) {
					return
				}
			}
		case <-timeout:
			if submissionDataCount != len(symbols) {
				t.Fatal("not all submission data received from websocket")
			}
		}
	}
}
