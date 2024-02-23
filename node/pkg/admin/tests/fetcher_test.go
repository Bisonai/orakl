//nolint:all
package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetcherStart(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	channel := testItems.mb.Subscribe("fetcher", 10)

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/start", nil)
	if err != nil {
		t.Fatalf("error starting fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher started")

	select {
	case msg := <-channel:
		if msg.From != "admin" || msg.To != "fetcher" || msg.Content.Command != "start" {
			t.Fatalf("unexpected message: %v", msg)
		}
	default:
		t.Fatalf("no message received on channel")
	}
}

func TestFetcherStop(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	channel := testItems.mb.Subscribe("fetcher", 10)

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/stop", nil)
	if err != nil {
		t.Fatalf("error stopping fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher stopped")

	select {
	case msg := <-channel:
		if msg.From != "admin" || msg.To != "fetcher" || msg.Content.Command != "stop" {
			t.Fatalf("unexpected message: %v", msg)
		}
	default:
		t.Fatalf("no message received on channel")
	}
}

func TestFetcherRefresh(t *testing.T) {
	ctx := context.Background()
	_cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer _cleanup()

	channel := testItems.mb.Subscribe("fetcher", 10)

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher refreshed")

	select {
	case msg := <-channel:
		if msg.From != "admin" || msg.To != "fetcher" || msg.Content.Command != "refresh" {
			t.Fatalf("unexpected message: %v", msg)
		}
	default:
		t.Fatalf("no message received on channel")
	}
}
