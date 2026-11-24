package wss

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSubscribeDelay(t *testing.T) {
	ws, err := NewWebsocketHelper(context.Background(), WithEndpoint("wss://example.invalid"))
	require.NoError(t, err)
	assert.Equal(t, time.Second, ws.subscribeWait(), "default")

	ws, err = NewWebsocketHelper(context.Background(), WithEndpoint("wss://example.invalid"), WithSubscribeDelay(100*time.Millisecond))
	require.NoError(t, err)
	assert.Equal(t, 100*time.Millisecond, ws.subscribeWait())
}
