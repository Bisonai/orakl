//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
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

func TestAggregatorInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockAggregator := aggregator.AggregatorInsertModel{
		Name: "test_aggregator_2",
	}

	readResultBefore, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators before: %v", err)
	}

	insertResult, err := PostRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", mockAggregator)
	if err != nil {
		t.Fatalf("error inserting aggregator: %v", err)
	}

	assert.Equal(t, insertResult.Name, mockAggregator.Name)

	readResultAfter, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more aggregators after insertion")

	// cleanup
	_, err = db.QueryRow[aggregator.AggregatorModel](context.Background(), aggregator.DeleteAggregatorById, map[string]any{"id": insertResult.Id})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestAggregatorGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators: %v", err)
	}

	assert.Greater(t, len(readResult), 0, "expected to have at least one aggregator")
}

func TestAggregatorGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/"+strconv.FormatInt(*testItems.tmpData.aggregator.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting aggregator by id: %v", err)
	}
	assert.Equal(t, readResult.Id, testItems.tmpData.aggregator.Id)
}

func TestAggregatorDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockAggregator := aggregator.AggregatorInsertModel{
		Name: "test_aggregator_2",
	}

	insertResult, err := PostRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", mockAggregator)
	if err != nil {
		t.Fatalf("error inserting aggregator: %v", err)
	}

	readResultBefore, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators before: %v", err)
	}

	deleteResult, err := DeleteRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/"+strconv.FormatInt(*insertResult.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deleting aggregator by id: %v", err)
	}

	assert.Equal(t, deleteResult.Id, insertResult.Id)

	readResultAfter, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators after: %v", err)
	}

	assert.Lessf(t, len(readResultAfter), len(readResultBefore), "expected to have less aggregators after deletion")

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

	activateResult, err := PostRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/activate/"+strconv.FormatInt(*testItems.tmpData.aggregator.Id, 10), nil)
	if err != nil {
		t.Fatalf("error activating aggregator: %v", err)
	}
	assert.True(t, activateResult.Active)

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

	deactivateResult, err := PostRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/deactivate/"+strconv.FormatInt(*testItems.tmpData.aggregator.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deactivating aggregator: %v", err)
	}
	assert.False(t, deactivateResult.Active)

}

func TestAggregatorSyncWithOraklConfig(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators before: %v", err)
	}

	_, err = RawPostRequest(testItems.app, "/api/v1/aggregator/sync/config", nil)
	if err != nil {
		t.Fatalf("error syncing aggregator with orakl config: %v", err)
	}

	readResultAfter, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators before: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more aggregators after syncing with orakl config")

	// cleanup
	_, err = db.QueryRow[aggregator.AggregatorModel](context.Background(), "DELETE FROM aggregators;", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestAggregatorAddFromOraklConfig(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators before: %v", err)
	}

	result, err := PostRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/sync/config/ADA-USDT", nil)
	if err != nil {
		t.Fatalf("error adding aggregator from orakl config: %v", err)
	}

	assert.Equal(t, result.Name, "ADA-USDT")

	readResultAfter, err := GetRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator", nil)
	if err != nil {
		t.Fatalf("error getting aggregators after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more aggregators after adding from orakl config")

	// cleanup
	_, err = db.QueryRow[aggregator.AggregatorModel](context.Background(), "DELETE FROM aggregators;", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}
