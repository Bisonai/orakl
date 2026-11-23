package wss

import (
	"context"

	"github.com/rs/zerolog"
)

// ReadErrorLevel is the level a custom read func (WithCustomReadFunc) logs a
// failed read at: Debug once ctx is done, else level. Run ends a read on
// purpose (stop, refresh, reconnect interval, keepalive drop) by cancelling
// or expiring its ctx before it closes the conn, so the resulting "context
// canceled" / "use of closed network connection" is not a failure; Run logs
// the reconnect itself.
func ReadErrorLevel(ctx context.Context, level zerolog.Level) zerolog.Level {
	if ctx.Err() != nil {
		return zerolog.DebugLevel
	}
	return level
}
