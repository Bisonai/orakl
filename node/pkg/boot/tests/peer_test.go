//nolint:all
package tests

import (
	"context"
	"testing"
	"time"

	adminTests "bisonai.com/miko/node/pkg/admin/tests"
	"bisonai.com/miko/node/pkg/boot"
	"bisonai.com/miko/node/pkg/boot/peer"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/miko/node/pkg/libp2p/utils"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx, t)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockHost1, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch())
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}

	mockHost2, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch())
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}

	url1, err := libp2pUtils.ExtractConnectionUrl(mockHost1)
	if err != nil {
		t.Fatalf("error extracting payload from host: %v", err)
	}
	url2, err := libp2pUtils.ExtractConnectionUrl(mockHost2)
	if err != nil {
		t.Fatalf("error extracting payload from host: %v", err)
	}

	mockPeer1 := peer.PeerInsertModel{
		Url: url1,
	}

	mockPeer2 := peer.PeerInsertModel{
		Url: url2,
	}

	syncResult, err := adminTests.PostRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer/sync", mockPeer1)
	if err != nil {
		t.Fatalf("error syncing peer: %v", err)
	}

	assert.Equal(t, len(syncResult), 1, "expected to have one pre-existing peer")

	syncResult, err = adminTests.PostRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer/sync", mockPeer2)
	if err != nil {
		t.Fatalf("error syncing peer: %v", err)
	}

	assert.Equal(t, len(syncResult), 2, "expected to have two pre-existing peers")

	// re-sync of a known peer is a no-op insert, still returns all peers
	syncResult, err = adminTests.PostRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer/sync", mockPeer1)
	if err != nil {
		t.Fatalf("error re-syncing peer: %v", err)
	}

	assert.Equal(t, len(syncResult), 3, "expected to have three pre-existing peers")
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx, t)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	h, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch()) // ephemeral port: a fixed one can collide with concurrent test runs
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}
	// closed explicitly below; cleanup covers FailNow paths (Close is idempotent)
	t.Cleanup(func() { _ = h.Close() })

	url, err := libp2pUtils.ExtractConnectionUrl(h)
	if err != nil {
		t.Fatalf("error extracting payload from host: %v", err)
	}

	res, _, err := testItems.store.Insert(ctx, url)
	if err != nil {
		t.Fatalf("error inserting peer: %v", err)
	}

	readResultBefore, err := testItems.store.List(ctx)
	if err != nil {
		t.Fatalf("error getting peers before: %v", err)
	}

	assert.Equal(t, res.Url, url, "expected to have the same url")

	bootHost, err := libp2pSetup.NewHost(ctx)
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}
	defer bootHost.Close()

	err = boot.RefreshJob(ctx, bootHost, testItems.store)
	if err != nil {
		t.Fatalf("error refreshing peers: %v", err)
	}

	readResultAfter, err := testItems.store.List(ctx)
	if err != nil {
		t.Fatalf("error getting peers after: %v", err)
	}

	// the unreachable sample peer is removed, the live host h is kept
	require.Len(t, readResultBefore, 2, "expected sample peer and h before refresh")
	require.Len(t, readResultAfter, 1, "expected only the live peer to remain")
	assert.Equal(t, url, readResultAfter[0].Url, "expected the live peer to be kept")

	h.Close()

	// the first refresh left bootHost connected to h; Connect is a no-op on an existing
	// connection, so wait until bootHost observes the close before probing h again (#3006)
	require.Eventually(t, func() bool {
		return bootHost.Network().Connectedness(h.ID()) != network.Connected
	}, 10*time.Second, 10*time.Millisecond, "bootHost still connected to closed host")

	err = boot.RefreshJob(ctx, bootHost, testItems.store)
	if err != nil {
		t.Fatalf("error refreshing peers: %v", err)
	}

	readResultFinal, err := testItems.store.List(ctx)
	if err != nil {
		t.Fatalf("error getting peers final: %v", err)
	}

	assert.Empty(t, readResultFinal, "expected the closed peer to be removed")
}
