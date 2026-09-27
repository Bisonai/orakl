package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/admin/config"
	"bisonai.com/miko/node/pkg/admin/feed"
	"github.com/stretchr/testify/assert"
)

// hermetic fixture: a JSON array matching []config.ConfigInsertModel, with more
// than one config and their feeds. It must not contain the tmp config/feed that
// setup() inserts, so the sync handler prunes them.
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
// host and delegates everything else to the wrapped transport. Installing it on
// http.DefaultTransport keeps TestConfigSync hermetic without any production
// hook: getConfigUrl() and the request package are unchanged, and the request
// package's client (no explicit Transport) falls through to http.DefaultTransport.
type configFixtureTransport struct {
	base http.RoundTripper
}

func (t configFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "config.orakl.network" {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(configSyncFixture)),
			Request:    req,
		}, nil
	}
	return t.base.RoundTrip(req)
}

func TestConfigSync(t *testing.T) {
	ctx := context.Background()

	// Intercept the live config.orakl.network fetch inside the sync handler.
	origTransport := http.DefaultTransport
	http.DefaultTransport = configFixtureTransport{base: origTransport}
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

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Greater(t, len(readResult), 1)

	// should remove previously inserted config and feed which doesn't exist in miko-config
	readTmpConfigResult, err := GetRequest[config.ConfigModel](testItems.app, "/api/v1/config/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Equal(t, config.ConfigModel{}, readTmpConfigResult)

	readTmpFeedResult, err := GetRequest[feed.FeedModel](testItems.app, "/api/v1/feed/"+strconv.Itoa(int(*testItems.tmpData.feed.ID)), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	expectedFeed := feed.FeedModel{
		Definition: json.RawMessage("null"),
	}
	assert.Equal(t, expectedFeed, readTmpFeedResult)
}

func TestConfigInsert(t *testing.T) {
	ctx := context.Background()
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

	insertResult, err := PostRequest[config.ConfigModel](testItems.app, "/api/v1/config", config.ConfigModel{
		Name:              "test",
		FetchInterval:     nil,
		AggregateInterval: nil,
		SubmitInterval:    nil,
	})
	if err != nil {
		t.Fatalf("error inserting config: %v", err)
	}
	assert.NotEqual(t, 0, insertResult.ID)

}

func TestConfigRead(t *testing.T) {
	ctx := context.Background()
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

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestConfigReadById(t *testing.T) {
	ctx := context.Background()
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

	readResult, err := GetRequest[config.ConfigModel](testItems.app, "/api/v1/config/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}
	assert.Equal(t, testItems.tmpData.config.ID, readResult.ID)
}

func TestConfigDeleteById(t *testing.T) {
	ctx := context.Background()
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

	deleted, err := DeleteRequest[config.ConfigModel](testItems.app, "/api/v1/config/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error deleting config: %v", err)
	}
	assert.Equal(t, testItems.tmpData.config.ID, deleted.ID)

	readResult, err := GetRequest[[]config.ConfigModel](testItems.app, "/api/v1/config", nil)
	if err != nil {
		t.Fatalf("error getting config: %v", err)
	}

	assert.Equal(t, 0, len(readResult))

}
