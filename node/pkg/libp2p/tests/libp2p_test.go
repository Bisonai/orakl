//nolint:all
package tests

import (
	"context"
	"fmt"
	"testing"

	"bisonai.com/orakl/node/pkg/boot"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/libp2p/setup"
	"bisonai.com/orakl/node/pkg/libp2p/utils"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestMakeHost(t *testing.T) {
	h, err := setup.MakeHost(10001)
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()
}

func TestMakePubsub(t *testing.T) {
	h, err := setup.MakeHost(10001)
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
	h, err := setup.MakeHost(10001)
	if err != nil {
		t.Fatalf("Failed to make host: %v", err)
	}
	defer h.Close()
	_, err = utils.GetHostAddress(h)
	if err != nil {
		t.Errorf("Failed to get host address: %v", err)
	}
}

func TestSetupFromBootApi(t *testing.T) {

	ctx := context.Background()
	go func() {
		err := boot.Run(ctx)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Failed to start boot server")
		}
	}()

	h1, _, err := setup.SetupFromBootApi(ctx, 10001)
	if err != nil {
		t.Errorf("Failed to setup from boot api: %v", err)
	}
	defer h1.Close()

	fmt.Println("h1: ", h1.ID())

	h2, _, err := setup.SetupFromBootApi(ctx, 10002)
	if err != nil {
		t.Errorf("Failed to setup from boot api: %v", err)
	}
	defer h2.Close()

	assert.Equal(t, network.Connected, h1.Network().Connectedness(h2.ID()))
	assert.Equal(t, network.Connected, h2.Network().Connectedness(h1.ID()))

	// cleanup db
	err = db.QueryWithoutResult(ctx, "DELETE FROM peers;", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}
