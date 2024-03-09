//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestAdapterInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockAdapter1 := adapter.AdapterInsertModel{
		Name: "test_adapter_2",
	}

	readResultBefore, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters before: %v", err)
	}

	insertResult, err := PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter", mockAdapter1)
	if err != nil {
		t.Fatalf("error inserting adapter: %v", err)
	}
	assert.Equal(t, insertResult.Name, mockAdapter1.Name)

	readResultAfter, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more adapters after insertion")

	//cleanup
	_, err = db.QueryRow[adapter.AdapterModel](context.Background(), adapter.DeleteAdapterById, map[string]any{"id": insertResult.Id})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestAdapterGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestAdapterReadDetailById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[adapter.AdapterDetailModel](testItems.app, "/api/v1/adapter/detail/"+strconv.FormatInt(*testItems.tmpData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting adapter detail: %v", err)
	}
	assert.Equal(t, readResult.Id, testItems.tmpData.adapter.Id)
	assert.NotEmpty(t, readResult.Feeds)
}

func TestAdapterGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/"+strconv.FormatInt(*testItems.tmpData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting adapter by id: %v", err)
	}
	assert.Equal(t, readResult.Id, testItems.tmpData.adapter.Id)
}

func TestAdapterDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockAdapter1 := adapter.AdapterInsertModel{
		Name: "test_adapter_2",
	}
	insertResult, err := PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter", mockAdapter1)
	if err != nil {
		t.Fatalf("error inserting adapter: %v", err)
	}

	readResultBefore, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters before: %v", err)
	}

	deleteResult, err := DeleteRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/"+strconv.FormatInt(*insertResult.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deleting adapter: %v", err)
	}
	assert.Equal(t, deleteResult.Id, insertResult.Id)

	readResultAfter, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters after: %v", err)
	}

	assert.Lessf(t, len(readResultAfter), len(readResultBefore), "expected to have less adapters after deletion")
}

func TestAdapterDeactivate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.FETCHER)
	waitForMessage(t, channel, bus.ADMIN, bus.FETCHER, bus.DEACTIVATE_FETCHER)

	deactivateResult, err := PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/deactivate/"+strconv.FormatInt(*testItems.tmpData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deactivating adapter: %v", err)
	}
	assert.False(t, deactivateResult.Active)
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
	activateResult, err := PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/activate/"+strconv.FormatInt(*testItems.tmpData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error activating adapter: %v", err)
	}
	assert.True(t, activateResult.Active)

}

func TestAdapterSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters after: %v", err)
	}

	_, err = RawPostRequest(testItems.app, "/api/v1/adapter/sync", nil)
	if err != nil {
		t.Fatalf("error syncing adapter: %v", err)
	}

	readResultAfter, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more adapters after insertion")

	// cleanup
	err = db.QueryWithoutResult(ctx, "DELETE FROM adapters", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}
