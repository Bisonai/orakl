package wss

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

// KeepaliveFunc sends one client keepalive (e.g. an exchange's ping message)
// on w, the writer bound to the current connection. It must return promptly
// once ctx is done: ending a connection waits for it.
type KeepaliveFunc func(ctx context.Context, w ConnWriter) error

// MaxKeepaliveFailures is the number of consecutive keepalive failures after
// which the helper drops the connection so that Run reconnects.
const MaxKeepaliveFailures = 3

var errKeepaliveFailed = errors.New("keepalive failed repeatedly")

// WithKeepalive makes the helper call fn every interval on each connection,
// from right after dial (so also while subscribing) until that connection
// ends. A failing fn
// is logged and retried on the next tick; MaxKeepaliveFailures failures in a
// row close the connection so Run reconnects. interval must be positive.
func WithKeepalive(interval time.Duration, fn KeepaliveFunc) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.KeepaliveInterval = interval
		c.KeepaliveFunc = fn
		c.keepaliveSet = true
	}
}

// PingTimeout is the longest a WithPingKeepalive ping waits for its pong.
const PingTimeout = 10 * time.Second

// WithPingKeepalive is WithKeepalive with a websocket protocol ping frame as
// the keepalive, for servers that drop connections without client traffic
// but have no app-level ping message. Each ping waits for its pong for
// min(interval, PingTimeout); a missing pong counts as a keepalive failure.
// Pongs are read by Run's read loop, so while subscribing (before that loop
// starts on the connection) a missing pong is not a failure.
func WithPingKeepalive(interval time.Duration) ConnectionOption {
	timeout := min(interval, PingTimeout)
	return WithKeepalive(interval, func(ctx context.Context, w ConnWriter) error {
		p, ok := w.(pinger)
		if !ok {
			return errNoPinger
		}
		return p.ping(ctx, timeout)
	})
}

// pinger is implemented by the conn-bound writer the keepalive receives.
type pinger interface {
	ping(ctx context.Context, timeout time.Duration) error
}

var errNoPinger = errors.New("keepalive writer cannot send pings")

// WithKeepaliveIntervalFunc lets the keepalive interval change per
// connection: intervalFn is called once at the start of every connection
// (after dial, so a custom dial func can update what it returns). A
// non-positive result falls back to the WithKeepalive interval. Requires
// WithKeepalive.
func WithKeepaliveIntervalFunc(intervalFn func() time.Duration) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.KeepaliveIntervalFunc = intervalFn
	}
}

// keepaliveInterval returns the interval for a new connection.
func (ws *WebsocketHelper) keepaliveInterval() time.Duration {
	if ws.keepaliveIntervalFn == nil {
		return ws.keepaliveFallback
	}
	interval := ws.keepaliveIntervalFn()
	if interval <= 0 {
		log.Warn().Str("endpoint", ws.Endpoint).Dur("interval", interval).Dur("fallback", ws.keepaliveFallback).Msg("non-positive keepalive interval, using fallback")
		return ws.keepaliveFallback
	}
	return interval
}

// runKeepalive calls the keepalive func on w every interval until ctx (the
// connection's ctx) is done. After MaxKeepaliveFailures consecutive failures
// it calls drop, which ends the connection.
func (ws *WebsocketHelper) runKeepalive(ctx context.Context, interval time.Duration, w ConnWriter, drop func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		err := ws.keepaliveFn(ctx, w)
		if ctx.Err() != nil {
			// the connection ended meanwhile; the next one runs its own keepalive
			return
		}
		if err == nil {
			failures = 0
			continue
		}

		failures++
		log.Warn().Err(err).Str("endpoint", ws.Endpoint).Int("failures", failures).Msg("websocket keepalive failed")
		if failures >= MaxKeepaliveFailures {
			drop()
			return
		}
	}
}
