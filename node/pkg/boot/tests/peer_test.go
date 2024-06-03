//nolint:all
package tests

import (
	"context"
	"strconv"
	"strings"
	"testing"

	adminTests "bisonai.com/orakl/node/pkg/admin/tests"
	"bisonai.com/orakl/node/pkg/boot"
	"bisonai.com/orakl/node/pkg/boot/peer"
	"bisonai.com/orakl/node/pkg/db"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/orakl/node/pkg/libp2p/utils"

	_peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestPeerInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockPeer1 := peer.PeerInsertModel{
		Ip:     "127.0.0.2",
		Port:   10002,
		HostId: "12DGKooWM8vWWqGPWWNCVPqb4tfqGrzx45W257GDBSeYbDSSLdef",
	}

	readResultBefore, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers before: %v", err)
	}

	insertResult, err := adminTests.PostRequest[peer.PeerModel](testItems.app, "/api/v1/peer", mockPeer1)
	if err != nil {
		t.Fatalf("error inserting peer: %v", err)
	}
	assert.Equal(t, insertResult.Ip, mockPeer1.Ip)

	readResultAfter, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more peers after insertion")

	//cleanup
	_, err = db.QueryRow[peer.PeerModel](ctx, peer.DeletePeerById, map[string]any{"id": insertResult.ID})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestPeerGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers: %v", err)
	}
	assert.Greater(t, len(readResult), 0, "expected to have at least one peer")
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockHost1, err := libp2pSetup.MakeHost(0)
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}

	mockHost2, err := libp2pSetup.MakeHost(0)
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}

	ip1, port1, hostId1, err := libp2pUtils.ExtractPayloadFromHost(mockHost1)
	if err != nil {
		t.Fatalf("error extracting payload from host: %v", err)
	}
	ip2, port2, hostId2, err := libp2pUtils.ExtractPayloadFromHost(mockHost2)
	if err != nil {
		t.Fatalf("error extracting payload from host: %v", err)
	}

	mockPeer1 := peer.PeerInsertModel{
		Ip:     ip1,
		Port:   port1,
		HostId: hostId1,
	}

	mockPeer2 := peer.PeerInsertModel{
		Ip:     ip2,
		Port:   port2,
		HostId: hostId2,
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

	h, err := libp2pSetup.MakeHost(10011)
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}

	pi := _peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	var addr multiaddr.Multiaddr
	for _, a := range pi.Addrs {
		if strings.Contains(a.String(), "127.0.0.1") {
			continue
		}
		addr = a
		break
	}

	splitted := strings.Split(addr.String(), "/")
	if len(splitted) < 5 {
		t.Fatalf("error splitting address: %v", splitted)
	}
	ip := splitted[2]
	port := splitted[4]

	portInt, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("error converting port to int: %v", err)
	}

	res, err := adminTests.PostRequest[peer.PeerModel](testItems.app, "/api/v1/peer", peer.PeerInsertModel{
		Ip:     ip,
		Port:   portInt,
		HostId: h.ID().String(),
	})
	if err != nil {
		t.Fatalf("error inserting peer: %v", err)
	}

	readResultBefore, err := adminTests.GetRequest[[]peer.PeerModel](testItems.app, "/api/v1/peer", nil)
	if err != nil {
		t.Fatalf("error getting peers before: %v", err)
	}

	assert.Equal(t, res.Ip, ip, "expected to have the same ip")

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
