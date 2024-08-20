package wss

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type WebsocketHelper struct {
	Conn              *websocket.Conn
	Endpoint          string
	Subscriptions     []any
	Proxy             string
	IsRunning         bool
	Compression       bool
	CustomDialFunc    *func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error)
	CustomReadFunc    *func(context.Context, *websocket.Conn) (map[string]interface{}, error)
	RequestHeaders    map[string]string
	ReadLimit         int64
	ReconnectInterval time.Duration
	InactivityTimeout time.Duration
	lastMessageTime   time.Time
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
