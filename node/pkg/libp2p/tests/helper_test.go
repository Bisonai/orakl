//nolint:all
package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/libp2p/helper"
	"bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()

	mb := bus.New(10)

	app := helper.New(mb, h)
	assert.NotNil(t, app)
	assert.Equal(t, mb, app.Bus)
	assert.Equal(t, h, app.Host)
}

func TestAppRunAndStop(t *testing.T) {
	ctx := context.Background()
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()
	mb := bus.New(10)

	libp2pHelper := helper.New(mb, h)
	assert.NotNil(t, libp2pHelper)
	err = libp2pHelper.Run(ctx)
	if err != nil {
		t.Errorf("Failed to run: %v", err)
	}
}

func TestAppGetPeerCount(t *testing.T) {
	ctx := context.Background()
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()
	mb := bus.New(10)

	libp2pHelper := helper.New(mb, h)
	err = libp2pHelper.Run(ctx)
	if err != nil {
		t.Errorf("Failed to run: %v", err)
	}
	assert.NotNil(t, libp2pHelper)

	msg := bus.Message{
		From: bus.ADMIN,
		To:   bus.LIBP2P,
		Content: bus.MessageContent{
			Command: bus.GET_PEER_COUNT,
			Args:    nil,
		},
		Response: make(chan bus.MessageResponse),
	}
	err = mb.Publish(msg)
	if err != nil {
		t.Errorf("Failed to publish msg: %v", err)
	}

	res := <-msg.Response
	assert.True(t, res.Success)
	assert.Equal(t, 0, res.Args["Count"].(int))

	h2, err := setup.NewHost(ctx)
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h2.Close()
	err = h.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()})
	if err != nil {
		t.Errorf("Failed to connect: %v", err)
	}

	err = mb.Publish(msg)
	if err != nil {
		t.Errorf("Failed to publish msg: %v", err)
	}

	res = <-msg.Response
	assert.True(t, res.Success)
	assert.Equal(t, 1, res.Args["Count"].(int))
}

func TestReconnectTriggerAfterDisconnection(t *testing.T) {
	t.Skip()
	ctx := context.Background()
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()
	mb := bus.New(10)

	libp2pHelper := helper.New(mb, h)
	err = libp2pHelper.Run(ctx)
	if err != nil {
		t.Errorf("Failed to run: %v", err)
	}

	h2, err := setup.NewHost(ctx)
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h2.Close()
	err = h.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()})
	if err != nil {
		t.Errorf("Failed to connect: %v", err)
	}

	h2.Close()

}
