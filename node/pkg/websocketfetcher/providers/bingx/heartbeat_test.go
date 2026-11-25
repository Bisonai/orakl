package bingx

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
	f := &BingxFetcher{}
	msg := map[string]any{"ping": "a1b2", "time": "2026-11-18T00:00:00.000+0800"}

	w := &wsstest.Writer{}
	require.NoError(t, f.handleMessage(wss.ContextWithWriter(context.Background(), w), msg))
	assert.Equal(t, []string{`{"pong":"a1b2","time":"2026-11-18T00:00:00.000+0800"}`}, w.Frames())

	assert.ErrorIs(t, f.handleMessage(context.Background(), msg), wss.ErrNoConnWriter)
}
