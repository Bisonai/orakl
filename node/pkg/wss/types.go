package wss

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type WebsocketHelper struct {
	// conn is the live connection, nil when not connected. Dial stores it and
	// Close swaps it out, so writers on other goroutines never race a reconnect.
	conn atomic.Pointer[websocket.Conn]
	// running guards Run against concurrent invocations.
	running atomic.Bool

	// keepalive (see WithKeepalive); keepaliveFn is nil when disabled
	keepaliveFn         KeepaliveFunc
	keepaliveFallback   time.Duration
	keepaliveIntervalFn func() time.Duration

	Endpoint          string
	Subscriptions     []any
	Proxy             string
	Compression       bool
	CustomDialFunc    *func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error)
	CustomReadFunc    *func(context.Context, *websocket.Conn) (map[string]interface{}, error)
	RequestHeaders    map[string]string
	ReadLimit         int64
	ReconnectInterval time.Duration
	InactivityTimeout time.Duration
}

type ConnectionConfig struct {
	Endpoint          string
	Proxy             string
	Subscriptions     []any
	Compression       bool
	DialFunc          func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error)
	ReadFunc          func(context.Context, *websocket.Conn) (map[string]interface{}, error)
	RequestHeaders    map[string]string
	ReadLimit         int64
	ReconnectInterval time.Duration
	InactivityTimeout time.Duration

	KeepaliveInterval     time.Duration
	KeepaliveFunc         KeepaliveFunc
	KeepaliveIntervalFunc func() time.Duration
	keepaliveSet          bool
}

type ConnectionOption func(*ConnectionConfig)

const DefaultReconnectInterval = 12 * time.Hour
const DefaultInactivityTimeout = 15 * time.Minute

func WithEndpoint(endpoint string) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Endpoint = endpoint
	}
}

func WithProxyUrl(proxyUrl string) ConnectionOption {
	return func(c *ConnectionConfig) {
		if proxyUrl == "" {
			return
		}
		c.Proxy = proxyUrl
	}
}

func WithSubscriptions(subscriptions []any) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Subscriptions = subscriptions
	}
}

func WithCustomDialFunc(dialFunc func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error)) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.DialFunc = dialFunc
	}
}

func WithCustomReadFunc(readFunc func(context.Context, *websocket.Conn) (map[string]interface{}, error)) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.ReadFunc = readFunc
	}
}

func WithCompressionMode() ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Compression = true
	}
}

func WithRequestHeaders(headers map[string]string) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.RequestHeaders = headers
	}
}

func WithReadLimit(readLimit int64) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.ReadLimit = readLimit
	}
}

func WithReconnectInterval(duration time.Duration) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.ReconnectInterval = duration
	}
}

func WithInactivityTimeout(duration time.Duration) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.InactivityTimeout = duration
	}
}

func NewWebsocketHelper(ctx context.Context, opts ...ConnectionOption) (*WebsocketHelper, error) {
	config := &ConnectionConfig{
		ReconnectInterval: DefaultReconnectInterval,
		InactivityTimeout: DefaultInactivityTimeout,
	}
	for _, opt := range opts {
		opt(config)
	}

	if config.Endpoint == "" {
		log.Error().Msg("endpoint is required")
		return nil, fmt.Errorf("endpoint is required")
	}

	// a non-positive window would expire every read immediately (endless reconnect loop)
	if config.ReconnectInterval <= 0 || config.InactivityTimeout <= 0 {
		log.Error().Msg("reconnect interval and inactivity timeout must be positive")
		return nil, fmt.Errorf("reconnect interval and inactivity timeout must be positive")
	}

	if config.keepaliveSet && config.KeepaliveInterval <= 0 {
		log.Error().Msg("keepalive interval must be positive")
		return nil, fmt.Errorf("keepalive interval must be positive")
	}
	if config.KeepaliveFunc == nil && (config.keepaliveSet || config.KeepaliveIntervalFunc != nil) {
		log.Error().Msg("keepalive func is required")
		return nil, fmt.Errorf("keepalive func is required")
	}

	if config.Subscriptions == nil {
		log.Warn().Msg("no subscriptions provided")
	}

	ws := &WebsocketHelper{
		Endpoint:          config.Endpoint,
		Subscriptions:     config.Subscriptions,
		Proxy:             config.Proxy,
		Compression:       config.Compression,
		RequestHeaders:    config.RequestHeaders,
		ReconnectInterval: config.ReconnectInterval,
		InactivityTimeout: config.InactivityTimeout,

		keepaliveFn:         config.KeepaliveFunc,
		keepaliveFallback:   config.KeepaliveInterval,
		keepaliveIntervalFn: config.KeepaliveIntervalFunc,
	}

	if config.DialFunc != nil {
		ws.CustomDialFunc = &config.DialFunc
	}

	if config.ReadFunc != nil {
		ws.CustomReadFunc = &config.ReadFunc
	}

	if config.ReadLimit > 0 {
		ws.ReadLimit = config.ReadLimit
	}

	return ws, nil
}
