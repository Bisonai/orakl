package lbank

import (
	"context"
	"testing"

	"bisonai.com/orakl/node/pkg/wss"
	"bisonai.com/orakl/node/pkg/wss/wsstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// the reply goes to the conn-bound writer with the same payload as before
// (f.Ws.Write)
func TestHeartbeatReply(t *testing.T) {
	f := &LbankFetcher{}
	msg := map[string]any{"action": "ping", "ping": "0ca8f854-7ba7-4341-9d86-d3327e52804e"}

	w := &wsstest.Writer{}
	require.NoError(t, f.handleMessage(wss.ContextWithWriter(context.Background(), w), msg))
	assert.Equal(t, []string{`{"action":"pong","pong":"0ca8f854-7ba7-4341-9d86-d3327e52804e"}`}, w.Frames())

	assert.ErrorIs(t, f.handleMessage(context.Background(), msg), wss.ErrNoConnWriter)
}
