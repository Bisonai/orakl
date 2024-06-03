//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/feed"
	"github.com/stretchr/testify/assert"
)

func TestFeedGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]feed.FeedModel](testItems.app, "/api/v1/feed", nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestFeedGetByConfigId(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]feed.FeedModel](testItems.app, "/api/v1/feed/config/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
	assert.Equal(t, testItems.tmpData.feed.ID, readResult[0].ID)
}

func TestFeedGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[feed.FeedModel](testItems.app, "/api/v1/feed/"+strconv.Itoa(int(*testItems.tmpData.feed.ID)), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Equal(t, testItems.tmpData.feed.ID, readResult.ID)
}
