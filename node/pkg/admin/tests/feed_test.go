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

func TestFeedGetByAdapterId(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]feed.FeedModel](testItems.app, "/api/v1/feed/adapter/"+strconv.FormatInt(*testItems.tempData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
	assert.Equal(t, testItems.tempData.feed.Id, readResult[0].Id)
}

func TestFeedGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[feed.FeedModel](testItems.app, "/api/v1/feed/"+strconv.FormatInt(*testItems.tempData.feed.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting feeds: %v", err)
	}
	assert.Equal(t, testItems.tempData.feed.Id, readResult.Id)
}
