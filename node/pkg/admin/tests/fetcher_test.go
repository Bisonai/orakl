//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"github.com/stretchr/testify/assert"
)

func TestFetcherStart(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.FETCHER)
	waitForMessage(t, channel, bus.ADMIN, bus.FETCHER, bus.START_FETCHER_APP)

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/start", nil)
	if err != nil {
		t.Fatalf("error starting fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher started: true")
}

func TestFetcherStop(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.FETCHER)
	waitForMessage(t, channel, bus.ADMIN, bus.FETCHER, bus.STOP_FETCHER_APP)

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/stop", nil)
	if err != nil {
		t.Fatalf("error stopping fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher stopped: true")
}

func TestFetcherRefresh(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.FETCHER)
	waitForMessage(t, channel, bus.ADMIN, bus.FETCHER, bus.REFRESH_FETCHER_APP)

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher refreshed: true")
}

func TestFetcherDeactivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.FETCHER)
	waitForMessage(t, channel, bus.ADMIN, bus.FETCHER, bus.DEACTIVATE_FETCHER)

	_, err = RawPostRequest(testItems.app, "/api/v1/fetcher/deactivate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error deactivating adapter: %v", err)
	}
}

func TestAdapterActivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.FETCHER)
	waitForMessage(t, channel, bus.ADMIN, bus.FETCHER, bus.ACTIVATE_FETCHER)

	// activate
	_, err = RawPostRequest(testItems.app, "/api/v1/fetcher/activate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error activating adapter: %v", err)
	}
}
