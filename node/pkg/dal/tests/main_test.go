//nolint:all
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/common/keys"
	"bisonai.com/miko/node/pkg/common/types"
	"bisonai.com/miko/node/pkg/dal/apiv2"
	"bisonai.com/miko/node/pkg/dal/collector"
	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/dal/utils/keycache"
	"bisonai.com/miko/node/pkg/dal/utils/stats"
	"bisonai.com/miko/node/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config = types.Config

type TestItems struct {
	App        *apiv2.ServerV2
	Collector  *collector.Collector
	Controller *hub.Hub
	TmpConfig  Config
	MockAdmin  *httptest.Server
	MockDal    *httptest.Server
	ApiKey     string
	StatsApp   *stats.StatsApp
}

func testPublishData(ctx context.Context, name string, submissionData aggregator.SubmissionData) error {
	return db.Publish(ctx, keys.SubmissionDataStreamKey(name), submissionData)
}

// publishAndAwait publishes submissionData and blocks until the collector has ingested it. The
// collector subscribes to redis asynchronously (Start spawns the subscriber in a goroutine), so a
// publish that races ahead of the subscription is silently dropped by redis pub/sub — a single
// fixed sleep after publishing is therefore inherently flaky. This re-publishes until the value
// lands (or the deadline passes); re-publishing the same timestamp is safe because the collector's
// timestamp compare-and-swap ignores duplicates once the value is stored.
func publishAndAwait(ctx context.Context, t *testing.T, testItems *TestItems, name string, submissionData aggregator.SubmissionData) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := testPublishData(ctx, name, submissionData); err != nil {
			t.Fatalf("error publishing sample submission data: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
		if _, err := testItems.Collector.GetLatestData(name); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("collector did not ingest %q within timeout", name)
		}
	}
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

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	testItems.ApiKey = "testApiKey"
	err := db.QueryWithoutResult(ctx, "INSERT INTO keys (key) VALUES (@newkey);", map[string]any{"newkey": "testApiKey"})
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert key in db")
		return nil, nil, err
	}

	mockAdminServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`[{
			"id": 13,
			"name": "test-aggregate",
			"fetchInterval": 15000,
			"aggregateInterval": 15000,
			"submitInterval": 15000}]`))
	}))

	testItems.TmpConfig = Config{
		ID:   13,
		Name: "test-aggregate",
	}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(ctx, 10*time.Minute)

	collector, err := collector.NewCollector(ctx, []types.Config{testItems.TmpConfig})
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return nil, nil, err
	}
	collector.Start(ctx)

	hub := hub.HubSetup(ctx, []types.Config{testItems.TmpConfig})
	go hub.Start(ctx, collector)

	statsApp := stats.NewStatsApp(ctx, stats.WithBulkLogsCopyInterval(1*time.Second))
	go statsApp.Run(ctx)

	server := apiv2.NewServer(collector, keyCache, hub, statsApp)

	mockDal := httptest.NewServer(server)

	testItems.App = server
	testItems.Collector = collector
	testItems.Controller = hub
	testItems.MockAdmin = mockAdminServer
	testItems.MockDal = mockDal
	testItems.StatsApp = statsApp

	return cleanup(ctx, testItems), testItems, nil
}

func cleanup(ctx context.Context, testItems *TestItems) func() error {
	return func() error {
		testItems.MockDal.Close()
		testItems.StatsApp.Stop()

		testItems.Collector.Stop()
		testItems.Controller = nil

		testItems.MockAdmin.Close()
		_ = db.QueryWithoutResult(ctx, "DELETE FROM keys", nil)
		return nil
	}
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	// setup
	code := m.Run()

	db.ClosePool()
	db.CloseRedis()

	// teardown
	os.Exit(code)
}
