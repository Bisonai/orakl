//nolint:all
package tests

import (
	"context"
	"testing"
	"time"

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.FETCHER || msg.Content.Command != bus.START_FETCHER_APP {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.FETCHER || msg.Content.Command != bus.STOP_FETCHER_APP {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()

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

	go func() {
		select {
		case msg := <-channel:
			if msg.From != bus.ADMIN || msg.To != bus.FETCHER || msg.Content.Command != bus.REFRESH_FETCHER_APP {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()

	result, err := RawPostRequest(testItems.app, "/api/v1/fetcher/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing fetcher: %v", err)
	}

	assert.Equal(t, string(result), "fetcher refreshed: true")
}
