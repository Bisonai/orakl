// //nolint:all
package reporter

import (
	"context"
	"os"
	"testing"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/stretchr/testify/assert"
)

func TestRunApp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := New()

	errChan := make(chan error, 1)
	defer close(errChan)

	go func() {
		err := app.Run(ctx)
		if err != nil {
			errChan <- err
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-errChan:
			t.Fatalf("error running reporter: %v", err)
		case <-ticker.C:
			if app.WsHelper != nil && app.WsHelper.IsRunning {
				return
			}
		}
	}
}

func TestRunMissingApiKey(t *testing.T) {
	os.Setenv("API_KEY", "")
	os.Setenv("DAL_WS_URL", "ws://test")
	os.Setenv("SUBMISSION_PROXY_CONTRACT", "0x123")
	ctx := context.Background()
	app := New()
	err := app.Run(ctx)
	assert.ErrorIs(t, err, errorSentinel.ErrReporterDalApiKeyNotFound)
}

func TestRunMissingWsUrl(t *testing.T) {
	os.Setenv("API_KEY", "test_api_key")
	os.Setenv("DAL_WS_URL", "")
	os.Setenv("SUBMISSION_PROXY_CONTRACT", "0x123")
	ctx := context.Background()
	app := New()
	err := app.Run(ctx)
	assert.NoError(t, err) // Should not return an error, should use default value
}

func TestRunMissingSubmissionProxyContract(t *testing.T) {
	os.Setenv("API_KEY", "test_api_key")
	os.Setenv("DAL_WS_URL", "ws://test")
	os.Setenv("SUBMISSION_PROXY_CONTRACT", "")
	ctx := context.Background()
	app := New()
	err := app.Run(ctx)
	assert.ErrorIs(t, err, errorSentinel.ErrReporterSubmissionProxyContractNotFound)
}

func TestWsDataHandling(t *testing.T) {
	ctx := context.Background()
	app := New()

	err := app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}

	configs, err := app.getConfigs(ctx)
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
				for _, config := range configs {
					if _, ok := app.LatestData.Load(config.Name); ok {
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
