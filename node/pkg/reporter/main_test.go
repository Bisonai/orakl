//nolint:all
package reporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/apiv2"
	"bisonai.com/orakl/node/pkg/dal/collector"
	dalcommon "bisonai.com/orakl/node/pkg/dal/common"
	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/utils/keystore"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/testutil/testenv"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
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

// dalAppEnv is what App.Run needs outside local mode: the DAL API key and
// endpoints it subscribes to, and the chain (plus reporter key) it reads the
// whitelist with.
var dalAppEnv = []string{"API_KEY", "DAL_REST_URL", "DAL_WS_URL", "SUBMISSION_PROXY_CONTRACT", "KAIA_PROVIDER_URL", "KAIA_REPORTER_PK"}

// dalKeyDir holds the mock DAL's SQLite key file; removed after the tests.
var dalKeyDir string

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
	os.Setenv("CONFIG_BASE_URL", configServer.URL)
	os.Setenv("CONFIG_FALLBACK_URL", "")

	var err error
	dalKeyDir, err = os.MkdirTemp("", "dal-keys-")
	if err != nil {
		panic(err)
	}

	stopRedis, err := testenv.StartRedis()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	code := m.Run()

	db.CloseRedis()
	stopRedis()
	configServer.Close()
	_ = os.RemoveAll(dalKeyDir)
	os.Exit(code)
}

func generateSampleSubmissionData(signerPK string, configId int32, value int64, timestamp time.Time, round int32, symbol string) (*aggregator.SubmissionData, error) {
	ctx := context.Background()
	sampleGlobalAggregate := aggregator.GlobalAggregate{
		ConfigID:  configId,
		Value:     value,
		Timestamp: timestamp,
		Round:     round,
	}

	signHelper, err := helper.NewSigner(ctx, helper.WithSignerPk(signerPK))
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

// mockDal is an in-process DAL: the real hub and apiv2 websocket server, fed
// by a collector that trusts a fixed whitelist instead of reading it on-chain
// and has no redis subscriptions (tests push to its OutgoingStream).
type mockDal struct {
	collector *collector.Collector
	wsURL     string
	apiKey    string
}

// newMockDal starts a mockDal serving config, whose collector accepts proofs
// signed by whitelist. It stops when ctx is done or the test ends.
func newMockDal(t *testing.T, ctx context.Context, config types.Config, whitelist []kaiacommon.Address) (*mockDal, error) {
	apiKey := "testApiKey"
	keyStore, err := keystore.Open(ctx, filepath.Join(dalKeyDir, "dal.db"))
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = keyStore.Close() })
	if err = keyStore.Upsert(ctx, apiKey, "test"); err != nil {
		return nil, err
	}

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(ctx, 10*time.Minute)

	// Only the fields used by Streams and IncomingDataToOutgoingData are set:
	// NewCollector would need a chain RPC for the whitelist and live redis
	// subscriptions, neither of which this test is about (the redis ingest
	// path is covered in pkg/dal/tests). With no chain reader, callers must
	// only convert proofs signed by whitelist: a rejected signer starts an
	// on-demand whitelist refresh against the nil reader, which panics.
	c := &collector.Collector{
		OutgoingStream: map[string]chan *dalcommon.OutgoingSubmissionData{
			config.Name: make(chan *dalcommon.OutgoingSubmissionData, collector.OutgoingStreamSize),
		},
		FeedHashes:      map[string][]byte{config.Name: crypto.Keccak256([]byte(config.Name))},
		Configs:         map[string]types.Config{config.Name: config},
		CachedWhitelist: whitelist,
	}

	h := hub.HubSetup(ctx, []types.Config{config})
	h.Start(ctx, c)
	// Closing the stream ends the hub's broadcaster for it.
	t.Cleanup(func() { c.RemoveSymbols([]string{config.Name}) })

	server := httptest.NewServer(apiv2.NewServer(c, keyCache, keyStore, h))
	t.Cleanup(server.Close)

	return &mockDal{
		collector: c,
		wsURL:     server.URL + "/ws",
		apiKey:    apiKey,
	}, nil
}
