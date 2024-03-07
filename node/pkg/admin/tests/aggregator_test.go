//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"
	"time"

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.AGGREGATOR || msg.Content.Command != bus.START_AGGREGATOR_APP {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.AGGREGATOR || msg.Content.Command != bus.STOP_AGGREGATOR_APP {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.AGGREGATOR || msg.Content.Command != bus.REFRESH_AGGREGATOR_APP {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.AGGREGATOR || msg.Content.Command != bus.ACTIVATE_AGGREGATOR {
				t.Errorf("expected to receive activate message from admin to aggregator")
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("No message received on channel")
		}
	}()

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.AGGREGATOR || msg.Content.Command != bus.DEACTIVATE_AGGREGATOR {
				t.Errorf("expected to receive deactivate message from admin to aggregator")
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("No message received on channel")
		}
	}()

	deactivateResult, err := PostRequest[aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/deactivate/"+strconv.FormatInt(*testItems.tmpData.aggregator.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deactivating aggregator: %v", err)
	}
	assert.False(t, deactivateResult.Active)

}

func TestAggregatorSyncWithAdapter(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	syncResult, err := PostRequest[[]aggregator.AggregatorModel](testItems.app, "/api/v1/aggregator/sync", nil)
	if err != nil {
		t.Fatalf("error syncing aggregator with adapter: %v", err)
	}
	assert.Greater(t, len(syncResult), 0, "expected to have at least one aggregator")
	assert.Equal(t, syncResult[0].Name, testItems.tmpData.adapter.Name)

	// cleanup
	_, err = db.QueryRow[aggregator.AggregatorModel](context.Background(), aggregator.DeleteAggregatorById, map[string]any{"id": syncResult[0].Id})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}
