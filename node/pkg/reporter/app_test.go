// //nolint:all
package reporter

import (
	"context"
	"testing"
	"time"
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

	err := app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}

	configs, err := fetchConfigs()
	if err != nil {
		t.Fatalf("error getting configs: %v", err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	timeout := time.After(5 * time.Second)
	submissionDataCount := 0

	for {
		select {
		case <-ticker.C:
			if app.WsHelper != nil && app.WsHelper.IsRunning {
				submissionDataCount = 0
				for _, config := range configs {
					if _, ok := app.LatestDataMap.Load(config.Name); ok {
						submissionDataCount++
					}
				}
				if submissionDataCount == len(configs) {

					return
				}
			}
		case <-timeout:
			if submissionDataCount != len(configs) {
				t.Fatal("not all submission data received from websocket")
			}
		}
	}
}
