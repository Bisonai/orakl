//nolint:all
package tests

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/libp2p/helper"
	"bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	h, err := setup.NewHost(context.Background())
	if err != nil {
		t.Errorf("Failed to make host: %v", err)
	}
	defer h.Close()

	mb := bus.New(10)

	_ = helper.New(mb, h)
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
	err = libp2pHelper.Run(ctx)
	if err != nil {
		t.Errorf("Failed to run: %v", err)
	}
	assert.True(t, libp2pHelper.IsRunning)

	libp2pHelper.Stop()
	assert.False(t, libp2pHelper.IsRunning)
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
}
