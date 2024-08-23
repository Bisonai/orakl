//nolint:all
package reporter

import (
	"context"
	"os"
	"testing"
	"time"

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

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
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
		GlobalAggregate: sampleGlobalAggregate,
		Proof:           proof,
	}, nil
}

func mockDalWsServer(ctx context.Context) (*wss.WebsocketHelper, *types.Config, []types.Config, error) {
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
		ID:                13,
		Name:              "test-aggregate",
		FetchInterval:     15000,
		AggregateInterval: 15000,
		SubmitInterval:    15000,
	}

	configs := []types.Config{tmpConfig}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		return nil, nil, nil, err
	}
	collector.Start(ctx)

	h := hub.HubSetup(ctx, configs)
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

	return conn, &tmpConfig, configs, nil
}
