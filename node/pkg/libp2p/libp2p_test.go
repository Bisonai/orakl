//nolint:all
package libp2p

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
)

func TestMakeHost(t *testing.T) {
	_, err := MakeHost(10001)
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
}

func TestMakePubsub(t *testing.T) {
	h, _ := MakeHost(10001)
	_, err := MakePubsub(context.Background(), h)
	if err != nil {
		t.Errorf("Failed to make pubsub: %v", err)
	}
}

func TestGetHostAddress(t *testing.T) {
	h, _ := MakeHost(10001)
	_, err := GetHostAddress(h)
	if err != nil {
		t.Errorf("Failed to get host address: %v", err)
	}
}

func TestInitDHT(t *testing.T) {
	h, _ := MakeHost(10001)
	_ = initDHT(context.Background(), h, "")
}

func TestDiscoverPeers(t *testing.T) {
	ctx := context.Background()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	b, _, _ := SetBootNode(ctx, 10003, "")
	h1, _ := MakeHost(10001)
	h2, _ := MakeHost(10002)

	defer h1.Close()
	defer h2.Close()

	h1.Connect(ctx, (*b).Peerstore().PeerInfo((*b).ID()))
	h2.Connect(ctx, (*b).Peerstore().PeerInfo((*b).ID()))

	go DiscoverPeers(context.Background(), h1, "test-discover-peers", (*b).Addrs()[0].String())
	err := DiscoverPeers(context.Background(), h2, "test-discover-peers", (*b).Addrs()[0].String())

	if err != nil {
		t.Errorf("Failed to discover peers: %v", err)
	}
}
