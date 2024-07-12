//nolint:all
package tests

import (
	"context"

	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"github.com/stretchr/testify/assert"
)

func TestGetPeerCount(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.LIBP2P)
	waitForMessageWithResponse(t, channel, bus.ADMIN, bus.LIBP2P, bus.GET_PEER_COUNT, map[string]any{"Count": 1})

	result, err := GetRequest[struct{ Count int }](testItems.app, "/api/v1/host/peercount", nil)
	if err != nil {
		t.Fatalf("error getting peercount: %v", err)
	}

	assert.Equal(t, 1, result.Count)
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	channel := testItems.mb.Subscribe(bus.LIBP2P)
	waitForMessage(t, channel, bus.ADMIN, bus.LIBP2P, bus.SYNC)

	result, err := RawPostRequest(testItems.app, "/api/v1/host/sync", nil)
	if err != nil {
		t.Fatalf("error sync libp2p host: %v", err)
	}

	assert.Equal(t, string(result), "libp2p synced")
}
