package wss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type wsConn struct {
	*websocket.Conn
}

type ConnectionConfig struct {
	Endpoint string
	ProxyUrl string
}

type ConnectionOption func(*ConnectionConfig)

func WithEndpoint(endpoint string) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Endpoint = endpoint
	}
}

func WithProxyUrl(proxyUrl string) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.ProxyUrl = proxyUrl
	}
}

func NewConnection(ctx context.Context, opts ...ConnectionOption) (*wsConn, error) {
	config := &ConnectionConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.Endpoint == "" {
		log.Error().Msg("endpoint is required")
		return nil, fmt.Errorf("endpoint is required")
	}

	dialOption := &websocket.DialOptions{}

	if config.ProxyUrl != "" {
		proxyURL, err := url.Parse(config.ProxyUrl)
		if err != nil {
			return nil, err
		}

		proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
		proxyTransport.Proxy = http.ProxyURL(proxyURL)

		dialOption = &websocket.DialOptions{
			HTTPClient: &http.Client{
				Transport: proxyTransport,
			},
		}
	}

	conn, _, err := websocket.Dial(ctx, config.Endpoint, dialOption)
	if err != nil {
		log.Error().Err(err).Msg("error opening websocket connection")
		return nil, err
	}
	return &wsConn{conn}, nil
}

func (ws *wsConn) Write(ctx context.Context, message interface{}) error {
	err := wsjson.Write(ctx, ws.Conn, message)
	if err != nil {
		log.Error().Err(err).Msg("error writing to websocket")
		return err
	}
	return nil
}

func (ws *wsConn) Read(ctx context.Context, ch chan interface{}) {
	for {
		var t interface{}
		err := wsjson.Read(ctx, ws.Conn, &t)
		if err != nil {
			log.Error().Err(err).Msg("error reading from websocket")
			break
		}
		ch <- t
	}
}

func (ws *wsConn) Close(ctx context.Context) error {
	err := ws.Conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		log.Error().Err(err).Msg("error closing websocket")
		return err
	}
	return nil
}
