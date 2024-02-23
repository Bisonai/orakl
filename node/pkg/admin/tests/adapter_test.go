//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestAdapterInsert(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

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
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	readResult, err := GetRequest[[]adapter.AdapterModel](testItems.app, "/api/v1/adapter", nil)
	if err != nil {
		t.Fatalf("error getting adapters: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestAdapterReadDetailById(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	readResult, err := GetRequest[adapter.AdapterDetailModel](testItems.app, "/api/v1/adapter/detail/"+strconv.FormatInt(*testItems.tempData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting adapter detail: %v", err)
	}
	assert.Equal(t, readResult.Id, testItems.tempData.adapter.Id)
	assert.NotEmpty(t, readResult.Feeds)
}

func TestAdapterGetById(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	readResult, err := GetRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/"+strconv.FormatInt(*testItems.tempData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting adapter by id: %v", err)
	}
	assert.Equal(t, readResult.Id, testItems.tempData.adapter.Id)
}

func TestAdapterDeleteById(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

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
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	channel := testItems.mb.Subscribe("fetcher", 10)

	deactivateResult, err := PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/deactivate/"+strconv.FormatInt(*testItems.tempData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deactivating adapter: %v", err)
	}
	assert.False(t, deactivateResult.Active)

	select {
	case msg := <-channel:
		if msg.From != "admin" || msg.To != "fetcher" || msg.Content.Command != "deactivate" {
			t.Errorf("Message did not match expected. Got %v", msg)
		}
	default:
		t.Errorf("No message received on channel")
	}
}

func TestAdapterActivate(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	channel := testItems.mb.Subscribe("fetcher", 10)

	//first deactivate before activate
	_, err = PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/deactivate/"+strconv.FormatInt(*testItems.tempData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deactivating adapter: %v", err)
	}
	<-channel

	// activate
	activateResult, err := PostRequest[adapter.AdapterModel](testItems.app, "/api/v1/adapter/activate/"+strconv.FormatInt(*testItems.tempData.adapter.Id, 10), nil)
	if err != nil {
		t.Fatalf("error activating adapter: %v", err)
	}
	assert.True(t, activateResult.Active)

	select {
	case msg := <-channel:
		if msg.From != "admin" || msg.To != "fetcher" || msg.Content.Command != "activate" {
			t.Errorf("Message did not match expected. Got %v", msg)
		}
	default:
		t.Errorf("No message received on channel")
	}
}
