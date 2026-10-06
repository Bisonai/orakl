//nolint:all
package reporter

import (
	"context"
	"os"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/apiv2"
	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/stats"
	"bisonai.com/miko/node/pkg/db"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/rs/zerolog"
)

// testConfigsFixture mirrors the shape of a real <chain>_configs.json entry
// (name, intervals, feeds) but is served locally so the reporter test suite is
// self-contained. The reporter only reads name/submitInterval; the extra fields
// document the real shape and are ignored on unmarshal.
const testConfigsFixture = `[
	{
		"id": 1,
		"name": "test-aggregate",
		"fetchInterval": 2000,
		"aggregateInterval": 3000,
		"submitInterval": 15000,
		"feeds": [
			{"name": "test-feed-0", "definition": {"url": "https://example.com/0"}}
		]
	},
	{
		"id": 2,
		"name": "test-aggregate-2",
		"fetchInterval": 2000,
		"aggregateInterval": 3000,
		"submitInterval": 15000,
		"feeds": [
			{"name": "test-feed-1", "definition": {"url": "https://example.com/1"}}
		]
	}
]`

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Serve the config bundle locally so the reporter tests never hit the live
	// config.orakl.network CDN (whose test_configs.json was removed in
	// orakl-config #208 -> 404). The handler is path-agnostic, so it keeps
	// working across the _configs.json -> _feeds.json rename (#2587).
	configServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testConfigsFixture))
	}))
	configBaseURL = configServer.URL

	code := m.Run()

	configServer.Close()
	os.Exit(code)
}

func generateSampleSubmissionData(configId int32, value int64, timestamp time.Time, round int32, symbol string) (*aggregator.SubmissionData, error) {
	tmpSignerPK := os.Getenv("SIGNER_PK")

	ctx := context.Background()
	sampleGlobalAggregate := aggregator.GlobalAggregate{
		ConfigID:  configId,
		Value:     value,
		Timestamp: timestamp,
		Round:     round,
	}

	signHelper, err := helper.NewSigner(ctx, helper.WithSignerPk(tmpSignerPK))
	if err != nil {
		return nil, err
	}

	rawProof, err := signHelper.MakeGlobalAggregateProof(value, timestamp, symbol)
	if err != nil {
		return nil, err
	}

	proof := aggregator.Proof{
		ConfigID: configId,
		Round:    round,
		Proof:    rawProof,
	}

	return &aggregator.SubmissionData{
		Symbol:          symbol,
		GlobalAggregate: sampleGlobalAggregate,
		Proof:           proof,
	}, nil
}

func mockDalWsServer(ctx context.Context) (*wss.WebsocketHelper, *types.Config, []string, error) {
	apiKey := "testApiKey"
	err := db.QueryWithoutResult(
		ctx,
		"INSERT INTO keys (key) VALUES (@newkey);",
		map[string]any{"newkey": apiKey},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	tmpConfig := types.Config{
		ID:   13,
		Name: "test-aggregate",
	}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(ctx, 10 * time.Minute)

	collector, err := collector.NewCollector(ctx, []types.Config{tmpConfig})
	if err != nil {
		return nil, nil, nil, err
	}
	collector.Start(ctx)

	h := hub.HubSetup(ctx, []types.Config{tmpConfig})
	go h.Start(ctx, collector)

	statsApp := stats.NewStatsApp(ctx, stats.WithBulkLogsCopyInterval(1*time.Second))
	go statsApp.Run(ctx)

	server := apiv2.NewServer(collector, keyCache, h, statsApp)

	mockDal := httptest.NewServer(server)

	headers := map[string]string{"X-API-Key": apiKey}

	conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint(mockDal.URL+"/ws"), wss.WithRequestHeaders(headers))
	if err != nil {
		return nil, nil, nil, err
	}

	return conn, &tmpConfig, []string{tmpConfig.Name}, nil
}
