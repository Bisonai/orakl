package crypto

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
	f := &CryptoDotComFetcher{}
	msg := map[string]any{"id": float64(1587523073344), "method": "public/heartbeat", "code": float64(0)}

	w := &wsstest.Writer{}
	require.NoError(t, f.handleMessage(wss.ContextWithWriter(context.Background(), w), msg))
	assert.Equal(t, []string{`{"id":1587523073344,"method":"public/respond-heartbeat"}`}, w.Frames())

	assert.ErrorIs(t, f.handleMessage(context.Background(), msg), wss.ErrNoConnWriter)
}
