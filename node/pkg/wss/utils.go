package wss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type WebsocketHelper struct {
	Conn          *websocket.Conn
	Endpoint      string
	Subscriptions []any
	Proxy         string
	mu            sync.Mutex
}

type ConnectionConfig struct {
	Endpoint      string
	Proxy         string
	Subscriptions []any
}

type ConnectionOption func(*ConnectionConfig)

func WithEndpoint(endpoint string) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Endpoint = endpoint
	}
}

func WithProxyUrl(proxyUrl string) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Proxy = proxyUrl
	}
}

func WithSubscriptions(subscriptions []any) ConnectionOption {
	return func(c *ConnectionConfig) {
		c.Subscriptions = subscriptions
	}
}

func NewWebsocketHelper(ctx context.Context, opts ...ConnectionOption) (*WebsocketHelper, error) {
	config := &ConnectionConfig{}
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

	return &WebsocketHelper{
		Endpoint:      config.Endpoint,
		Subscriptions: config.Subscriptions,
		Proxy:         config.Proxy,
		mu:            sync.Mutex{},
	}, nil
}

func (ws *WebsocketHelper) Dial(ctx context.Context) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	dialOption := &websocket.DialOptions{}
	if ws.Proxy != "" {
		proxyURL, err := url.Parse(ws.Proxy)
		if err != nil {
			return err
		}

		proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
		proxyTransport.Proxy = http.ProxyURL(proxyURL)

		dialOption = &websocket.DialOptions{
			HTTPClient: &http.Client{
				Transport: proxyTransport,
			},
		}
	}

	conn, _, err := websocket.Dial(ctx, ws.Endpoint, dialOption)
	if err != nil {
		log.Error().Err(err).Msg("error opening websocket connection")
		return err
	}
	ws.Conn = conn
	return nil
}

func (ws *WebsocketHelper) Run(ctx context.Context, router func(context.Context, map[string]any) error) {
	dialJob := func() error {
		return ws.Dial(ctx)
	}

	subscribeJob := func() error {
		for _, subscription := range ws.Subscriptions {
			err := ws.Write(ctx, subscription)
			if err != nil {
				log.Error().Err(err).Msg("error writing subscription to websocket")
				return err
			}
		}
		return nil
	}

	for {
		err := retrier.Retry(dialJob, 3, 1, 10)
		if err != nil {
			log.Error().Err(err).Msg("error dialing websocket")
			break
		}

		err = retrier.Retry(subscribeJob, 3, 1, 10)
		if err != nil {
			log.Error().Err(err).Msg("error subscribing to websocket")
			break
		}

		for {
			var data map[string]any
			ws.mu.Lock()
			err := wsjson.Read(ctx, ws.Conn, &data)
			ws.mu.Unlock()
			if err != nil {
				log.Error().Err(err).Msg("error reading from websocket")
				break
			}
			err = router(ctx, data)
			if err != nil {
				log.Error().Err(err).Msg("error processing message")
			}
		}
		ws.mu.Lock()
		ws.Close()
		ws.mu.Unlock()
	}
}

func (ws *WebsocketHelper) Write(ctx context.Context, message interface{}) error {
	err := wsjson.Write(ctx, ws.Conn, message)
	if err != nil {
		log.Error().Err(err).Msg("error writing to websocket")
		return err
	}
	return nil
}

func (ws *WebsocketHelper) Read(ctx context.Context, ch chan any) error {
	for {
		var t any
		err := wsjson.Read(ctx, ws.Conn, &t)
		if err != nil {
			log.Error().Err(err).Msg("error reading from websocket")
			return err
		}
		ch <- t
	}
}

func (ws *WebsocketHelper) Close() error {
	err := ws.Conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		log.Error().Err(err).Msg("error closing websocket")
		return err
	}
	return nil
}
