//nolint:all
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/collector"
	"bisonai.com/orakl/node/pkg/dal/utils/initializer"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type TestItems struct {
	App        *fiber.App
	Collector  *collector.Collector
	Controller *api.Hub
	TmpConfig  types.Config
	MockAdmin  *httptest.Server
	ApiKey     string
}

func testPublishData(ctx context.Context, submissionData aggregator.SubmissionData) error {
	return db.Publish(ctx, keys.SubmissionDataStreamKey(submissionData.GlobalAggregate.ConfigID), submissionData)
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

	testItems.TmpConfig = types.Config{
		ID:                13,
		Name:              "test-aggregate",
		FetchInterval:     15000,
		AggregateInterval: 15000,
		SubmitInterval:    15000,
	}

	configs := []types.Config{testItems.TmpConfig}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return nil, nil, err
	}

	hub := api.HubSetup(ctx, configs)

	app, err := initializer.Setup(ctx, collector, hub, keyCache)
	if err != nil {
		return nil, nil, err
	}
	testItems.App = app
	testItems.Collector = collector
	testItems.Controller = hub
	testItems.MockAdmin = mockAdminServer

	v1 := app.Group("")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Node DAL API")
	})
	api.Routes(v1)

	return cleanup(ctx, testItems), testItems, nil
}

func cleanup(ctx context.Context, testItems *TestItems) func() error {
	return func() error {
		err := testItems.App.Shutdown()
		if err != nil {
			log.Error().Err(err).Msg("error shutting down app")
			return err
		}

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
