package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/nodeconfig"
	"github.com/stretchr/testify/assert"
)

// hermetic fixture: a JSON array matching []nodeconfig.Entry, with more than
// one config and their feeds.
const configSyncFixture = `[
	{
		"name": "test-aggregate-0",
		"fetchInterval": 2000,
		"aggregateInterval": 3000,
		"submitInterval": 15000,
		"decimals": 8,
		"feeds": [
			{"name": "test-feed-0", "definition": {"url": "https://example.com/0"}}
		]
	},
	{
		"name": "test-aggregate-1",
		"fetchInterval": 2000,
		"aggregateInterval": 3000,
		"submitInterval": 15000,
		"decimals": 8,
		"feeds": [
			{"name": "test-feed-1", "definition": {"url": "https://example.com/1"}}
		]
	}
]`

// configFixtureTransport serves configSyncFixture for requests to the config
// host and fails any other outbound request, so TestConfigSync makes no real
// network egress. Installing it on http.DefaultTransport keeps the test hermetic
// without any production hook: nodeconfig.URL() and the request package are
// unchanged, and the request package's client (no explicit Transport) falls
// through to http.DefaultTransport. If nodeconfig.URL() ever stops resolving to
// config.orakl.network, the request errors here instead of quietly hitting the
// live CDN.
type configFixtureTransport struct{}

func (t configFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "config.orakl.network" {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(configSyncFixture)),
			Request:    req,
		}, nil
	}
	return nil, fmt.Errorf("unexpected outbound request in hermetic test: %s", req.URL)
}

func TestConfigSync(t *testing.T) {
	ctx := context.Background()

	// Intercept the live config.orakl.network fetch inside the sync handler.
	origTransport := http.DefaultTransport
	http.DefaultTransport = configFixtureTransport{}
	defer func() { http.DefaultTransport = origTransport }()

	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		err = cleanup()
		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	_, err = RawPostRequest(testItems.app, "/api/v1/config/sync", nil)
	if err != nil {
		t.Fatalf("error syncing config: %v", err)
	}

	readResult, err := GetRequest[[]nodeconfig.Config](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	if assert.Len(t, readResult, 2) {
		assert.Equal(t, "test-aggregate-0", readResult[0].Name)
		assert.Equal(t, int32(3000), readResult[0].AggregateInterval)
		assert.Equal(t, "test-aggregate-1", readResult[1].Name)
		assert.NotEqual(t, readResult[0].ID, readResult[1].ID)
	}
	feeds := nodeconfig.Default().Feeds(readResult[0].ID)
	if assert.Len(t, feeds, 1) {
		assert.Equal(t, "test-feed-0", feeds[0].Name)
	}
}
