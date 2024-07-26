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
	ctx := context.Background()
	cleanUp, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanUp(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	app := New()
	err = app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}
}

func TestRunMissingApiKey(t *testing.T) {
	ctx := context.Background()
	cleanUp, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanUp(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()
	app := New()

	apiKey := os.Getenv("API_KEY")
	os.Setenv("API_KEY", "")

	err = app.Run(ctx)
	os.Setenv("API_KEY", apiKey)

	assert.ErrorIs(t, err, errorSentinel.ErrReporterDalApiKeyNotFound)
}

func TestRunMissingWsUrl(t *testing.T) {
	ctx := context.Background()
	cleanUp, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanUp(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()
	app := New()

	dalWsEndpoint := os.Getenv("DAL_WS_URL")
	os.Setenv("DAL_WS_URL", "")

	err = app.Run(ctx)
	os.Setenv("DAL_WS_URL", dalWsEndpoint)

	assert.NoError(t, err)
}

func TestRunMissingSubmissionProxyContract(t *testing.T) {
	ctx := context.Background()
	cleanUp, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanUp(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()
	app := New()

	submissionProxy := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	os.Setenv("SUBMISSION_PROXY_CONTRACT", "")

	err = app.Run(ctx)
	os.Setenv("SUBMISSION_PROXY_CONTRACT", submissionProxy)

	assert.ErrorIs(t, err, errorSentinel.ErrReporterSubmissionProxyContractNotFound)
}

func TestWsDataHandling(t *testing.T) {
	ctx := context.Background()
	cleanUp, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanUp(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()
	app := New()

	err = app.Run(ctx)
	if err != nil {
		t.Fatalf("error running reporter: %v", err)
	}

	configs, err := getConfigs(ctx)
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
