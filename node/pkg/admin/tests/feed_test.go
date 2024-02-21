//nolint:all
package tests

import (
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/feed"
	"github.com/stretchr/testify/assert"
)

func TestFeedGet(t *testing.T) {
	app, err := setup()
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	defer app.Shutdown()

	readResult, err := GetRequest[[]feed.FeedModel](app, "/api/v1/feed", nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestFeedGetByAdapterId(t *testing.T) {
	app, err := setup()
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	defer app.Shutdown()

	readResult, err := GetRequest[[]feed.FeedModel](app, "/api/v1/feed/adapter/"+strconv.FormatInt(*insertedAdapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
	assert.Equal(t, insertedFeed.Id, readResult[0].Id)
}

func TestFeedGetById(t *testing.T) {
	app, err := setup()
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	defer app.Shutdown()

	readResult, err := GetRequest[feed.FeedModel](app, "/api/v1/feed/"+strconv.FormatInt(*insertedFeed.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Equal(t, insertedFeed.Id, readResult.Id)
}
