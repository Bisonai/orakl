//nolint:all
package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/miko/node/pkg/admin/proxy"
	"bisonai.com/miko/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestProxyInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockProxy := proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "test_host",
		Port:     8080,
		Location: nil,
	}

	readResultBefore, err := GetRequest[[]proxy.ProxyModel](testItems.app, "/api/v1/proxy", nil)
	if err != nil {
		t.Fatalf("error getting proxies before: %v", err)
	}

	insertResult, err := PostRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy", mockProxy)
	if err != nil {
		t.Fatalf("error inserting proxy: %v", err)
	}
	assert.Equal(t, insertResult.Protocol, mockProxy.Protocol)
	assert.Equal(t, insertResult.Host, mockProxy.Host)
	assert.Equal(t, insertResult.Port, mockProxy.Port)

	readResultAfter, err := GetRequest[[]proxy.ProxyModel](testItems.app, "/api/v1/proxy", nil)
	if err != nil {
		t.Fatalf("error getting proxies after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more proxies after insertion")

	_, err = db.QueryRow[proxy.ProxyModel](context.Background(), proxy.DeleteProxyById, map[string]any{"id": insertResult.ID})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestProxyGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]proxy.ProxyModel](testItems.app, "/api/v1/proxy", nil)
	if err != nil {
		t.Fatalf("error getting proxies: %v", err)
	}

	assert.Greater(t, len(readResult), 0)
}

func TestProxyGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultById, err := GetRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy/"+strconv.Itoa(int(*testItems.tmpData.proxy.ID)), nil)
	if err != nil {
		t.Fatalf("error getting proxy by id: %v", err)
	}

	assert.Equal(t, readResultById.ID, testItems.tmpData.proxy.ID)
}

func TestProxyUpdateById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockProxy := proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "test_host",
		Port:     8080,
		Location: nil,
	}

	insertResult, err := PostRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy", mockProxy)
	if err != nil {
		t.Fatalf("error inserting proxy: %v", err)
	}

	mockProxyUpdate := proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "test_host_update",
		Port:     8080,
		Location: nil,
	}

	updateResult, err := PatchRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy/"+strconv.Itoa(int(*insertResult.ID)), mockProxyUpdate)
	if err != nil {
		t.Fatalf("error updating proxy: %v", err)
	}

	assert.Equal(t, updateResult.Host, mockProxyUpdate.Host)

	_, err = db.QueryRow[proxy.ProxyModel](context.Background(), proxy.DeleteProxyById, map[string]any{"id": insertResult.ID})
	if err != nil {
		t.Fatalf("error deleting proxy: %v", err)
	}
}

func TestProxyDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockProxy := proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "test_host",
		Port:     8080,
		Location: nil,
	}

	insertResult, err := PostRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy", mockProxy)
	if err != nil {
		t.Fatalf("error inserting proxy: %v", err)
	}

	_, err = DeleteRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy/"+strconv.Itoa(int(*insertResult.ID)), nil)
	if err != nil {
		t.Fatalf("error deleting proxy: %v", err)
	}

	result, err := GetRequest[proxy.ProxyModel](testItems.app, "/api/v1/proxy/"+strconv.Itoa(int(*insertResult.ID)), nil)
	if err != nil {
		t.Fatalf("error getting proxy by id: %v", err)
	}
	assert.Equal(t, result, *new(proxy.ProxyModel), "expected to get nil result after deletion")
}
