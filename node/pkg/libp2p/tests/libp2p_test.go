//nolint:all
package tests

import (
	"context"
	"testing"

	"bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/libp2p/utils"
)

func TestMakeHost(t *testing.T) {
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()
}

func TestMakePubsub(t *testing.T) {
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Fatalf("Failed to make host: %v", err)
	}
	defer h.Close()

	_, err = setup.MakePubsub(context.Background(), h)
	if err != nil {
		t.Errorf("Failed to make pubsub: %v", err)
	}
}

func TestGetHostAddress(t *testing.T) {
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Fatalf("Failed to make host: %v", err)
	}
	defer h.Close()
	_, err = utils.GetHostAddress(h)
	if err != nil {
		t.Errorf("Failed to get host address: %v", err)
	}
}

func TestReplaceIp(t *testing.T) {
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Fatalf("Failed to make host: %v", err)
	}
	defer h.Close()

	url, err := utils.ExtractConnectionUrl(h)
	if err != nil {
		t.Fatalf("Failed to extract connection url: %v", err)
	}

	result, err := utils.ReplaceIpFromUrl(url, "127.0.0.1")
	if err != nil {
		t.Errorf("Failed to replace ip: %v", err)
	}

	if url == result {
		t.Errorf("Failed to replace ip: %v", err)
	}
}
