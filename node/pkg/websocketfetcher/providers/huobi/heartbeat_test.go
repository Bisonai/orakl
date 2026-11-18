package huobi

import (
	"context"
	"testing"

	"bisonai.com/miko/node/pkg/wss"
	"bisonai.com/miko/node/pkg/wss/wsstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// the reply goes to the conn-bound writer with the same payload as before
// (f.Ws.Write)
func TestHeartbeatReply(t *testing.T) {
	f := &HuobiFetcher{}
	msg := map[string]any{"ping": float64(1763424000123)}

	w := &wsstest.Writer{}
	require.NoError(t, f.handleMessage(wss.ContextWithWriter(context.Background(), w), msg))
	assert.Equal(t, []string{`{"pong":1763424000123}`}, w.Frames())

	assert.ErrorIs(t, f.handleMessage(context.Background(), msg), wss.ErrNoConnWriter)
}
