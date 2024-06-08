//nolint:all
package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/libp2p/setup"
	"bisonai.com/orakl/node/pkg/libp2p/utils"
)

func TestMakeHost(t *testing.T) {
	h, err := setup.NewHost(context.Background(), setup.WithHolePunch(), setup.WithPort(10001))
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()
}

func TestMakePubsub(t *testing.T) {
	h, err := setup.NewHost(context.Background(), setup.WithHolePunch(), setup.WithPort(10001))
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
	h, err := setup.NewHost(context.Background(), setup.WithHolePunch(), setup.WithPort(10001))
	if err != nil {
		t.Fatalf("Failed to make host: %v", err)
	}
	defer h.Close()
	_, err = utils.GetHostAddress(h)
	if err != nil {
		t.Errorf("Failed to get host address: %v", err)
	}
}
