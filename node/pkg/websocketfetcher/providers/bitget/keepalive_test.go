package bitget

import (
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/wss/wsstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payload and interval as sent by the provider's own ping goroutine before
// the shared keepalive
func TestKeepalivePing(t *testing.T) {
	w := &wsstest.Writer{}
	require.NoError(t, ping(context.Background(), w))
	assert.Equal(t, []string{"ping"}, w.Frames())
	assert.Equal(t, 30*time.Second, PingInterval)
}
