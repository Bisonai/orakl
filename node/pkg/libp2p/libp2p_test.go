package libp2p

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"
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
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	h1, _ := MakeHost(10001)
	h2, _ := MakeHost(10002)

	defer h1.Close()
	defer h2.Close()

	go DiscoverPeers(context.Background(), h1, "test-discover-peers", h2.Addrs()[0].String())
	DiscoverPeers(context.Background(), h2, "test-discover-peers", h1.Addrs()[0].String())

	str := buf.String()
	if !strings.Contains(str, "Peer discovery complete") {
		t.Errorf("Expected 'Peer discovery complete' but it was not found")
	}
}
