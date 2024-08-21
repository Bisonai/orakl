//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/miko/node/pkg/admin/providerUrl"
	"bisonai.com/miko/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestProviderUrlInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	testChainId := 1

	mockProviderUrl1 := providerUrl.ProviderUrlInsertModel{
		ChainId: &testChainId,
		Url:     "test_provider_url_1",
	}

	readResultBefore, err := GetRequest[[]providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url", nil)
	if err != nil {
		t.Fatalf("error getting provider urls before: %v", err)
	}

	_, err = PostRequest[providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url", mockProviderUrl1)
	if err != nil {
		t.Fatalf("error inserting provider url: %v", err)
	}

	readResultAfter, err := GetRequest[[]providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url", nil)
	if err != nil {
		t.Fatalf("error getting provider urls after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more provider urls after insertion")

	//cleanup
	err = db.QueryWithoutResult(ctx, "DELETE FROM provider_urls;", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestProviderUrlGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url", nil)
	if err != nil {
		t.Fatalf("error getting provider urls: %v", err)
	}

	assert.Greater(t, len(readResult), 0)
}

func TestProviderUrlGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url/"+strconv.Itoa(int(*testItems.tmpData.providerUrl.ID)), nil)
	if err != nil {
		t.Fatalf("error getting provider urls: %v", err)
	}

	assert.Equal(t, testItems.tmpData.providerUrl, readResult)

}

func TestProviderDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url", nil)
	if err != nil {
		t.Fatalf("error getting provider urls before: %v", err)
	}

	_, err = db.QueryRow[providerUrl.ProviderUrlModel](context.Background(), providerUrl.DeleteProviderUrlById, map[string]interface{}{"id": testItems.tmpData.providerUrl.ID})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}

	readResultAfter, err := GetRequest[[]providerUrl.ProviderUrlModel](testItems.app, "/api/v1/provider-url", nil)
	if err != nil {
		t.Fatalf("error getting provider urls after: %v", err)
	}

	assert.Less(t, len(readResultAfter), len(readResultBefore))
}
