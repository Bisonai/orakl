//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"github.com/stretchr/testify/assert"
)

func TestAggregatorStart(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.AGGREGATOR)
	waitForMessage(t, channel, bus.ADMIN, bus.AGGREGATOR, bus.START_AGGREGATOR_APP)

	result, err := RawPostRequest(testItems.app, "/api/v1/aggregator/start", nil)
	if err != nil {
		t.Fatalf("error starting aggregator: %v", err)
	}

	assert.Equal(t, string(result), "aggregator started")
}

func TestAggregatorStop(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.AGGREGATOR)
	waitForMessage(t, channel, bus.ADMIN, bus.AGGREGATOR, bus.STOP_AGGREGATOR_APP)

	result, err := RawPostRequest(testItems.app, "/api/v1/aggregator/stop", nil)
	if err != nil {
		t.Fatalf("error stopping aggregator: %v", err)
	}

	assert.Equal(t, string(result), "aggregator stopped")

}

func TestAggregatorRefresh(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.AGGREGATOR)
	waitForMessage(t, channel, bus.ADMIN, bus.AGGREGATOR, bus.REFRESH_AGGREGATOR_APP)

	result, err := RawPostRequest(testItems.app, "/api/v1/aggregator/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing aggregator: %v", err)
	}

	assert.Equal(t, string(result), "aggregator refreshed")

}

func TestAggregatorActivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.AGGREGATOR)
	waitForMessage(t, channel, bus.ADMIN, bus.AGGREGATOR, bus.ACTIVATE_AGGREGATOR)

	_, err = RawPostRequest(testItems.app, "/api/v1/aggregator/activate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error activating aggregator: %v", err)
	}

}

func TestAggregatorDeactivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.AGGREGATOR)
	waitForMessage(t, channel, bus.ADMIN, bus.AGGREGATOR, bus.DEACTIVATE_AGGREGATOR)

	_, err = RawPostRequest(testItems.app, "/api/v1/aggregator/deactivate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error deactivating aggregator: %v", err)
	}
}
