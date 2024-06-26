//nolint:all
package tests

import (
	"context"
	"testing"

	adminTests "bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/boot"
	"bisonai.com/orakl/node/pkg/boot/peer"
	"bisonai.com/orakl/node/pkg/db"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/orakl/node/pkg/libp2p/utils"

	"github.com/stretchr/testify/assert"
)

func TestSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
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

	//cleanup
	err = db.QueryWithoutResult(ctx, "DELETE FROM peers;", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestRefresh(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	h, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch(), libp2pSetup.WithPort(10010))
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}

	url, err := libp2pUtils.ExtractConnectionUrl(h)
	if err != nil {
		t.Fatalf("error extracting payload from host: %v", err)
	}

	res, err := adminTests.PostRequest[peer.PeerModel](testItems.app, "/api/v1/peer", peer.PeerInsertModel{
		Url: url,
	})
	if err != nil {
		t.Fatalf("error inserting peer: %v", err)
	}

	readResultBefore, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers before: %v", err)
	}

	assert.Equal(t, res.Url, url, "expected to have the same url")

	err = boot.RefreshJob(ctx)
	if err != nil {
		t.Fatalf("error refreshing peers: %v", err)
	}

	readResultAfter, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers after: %v", err)
	}

	assert.Less(t, len(readResultAfter), len(readResultBefore), "expected to have less peers after refresh")

	h.Close()

	err = boot.RefreshJob(ctx)
	if err != nil {
		t.Fatalf("error refreshing peers: %v", err)
	}

	readResultFinal, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers final: %v", err)
	}

	assert.Less(t, len(readResultFinal), len(readResultAfter), "expected to have less peers after refresh")
}
