package wss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type WebsocketHelper struct {
	Conn           *websocket.Conn
	Endpoint       string
	Subscriptions  []any
	Proxy          string
	IsRunning      bool
	Compression    bool
	CustomDialFunc *func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error)
	CustomReadFunc *func(context.Context, *websocket.Conn) (map[string]interface{}, error)
	RequestHeaders map[string]string
	ReadLimit      int64
}

type ConnectionConfig struct {
	Endpoint       string
	Proxy          string
	Subscriptions  []any
	Compression    bool
	DialFunc       func(context.Context, string, *websocket.DialOptions) (*websocket.Conn, *http.Response, error)
	ReadFunc       func(context.Context, *websocket.Conn) (map[string]interface{}, error)
	RequestHeaders map[string]string
	ReadLimit      int64
}

type ConnectionOption func(*ConnectionConfig)

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

	ws := &WebsocketHelper{
		Endpoint:       config.Endpoint,
		Subscriptions:  config.Subscriptions,
		Proxy:          config.Proxy,
		Compression:    config.Compression,
		RequestHeaders: config.RequestHeaders,
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

func (ws *WebsocketHelper) Dial(ctx context.Context) error {
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

	if len(ws.RequestHeaders) > 0 {
		dialOption.HTTPHeader = http.Header{}
		for key, value := range ws.RequestHeaders {
			dialOption.HTTPHeader.Add(key, value)
		}
	}

	if ws.Compression {
		dialOption.CompressionMode = websocket.CompressionContextTakeover
	}

	dialFunc := websocket.Dial
	if ws.CustomDialFunc != nil {
		dialFunc = *ws.CustomDialFunc
	}
	conn, _, err := dialFunc(ctx, ws.Endpoint, dialOption)
	if err != nil {
		log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error opening websocket connection")
		return err
	}

	if ws.ReadLimit > 0 {
		conn.SetReadLimit(ws.ReadLimit)
	}

	ws.Conn = conn
	return nil
}

func (ws *WebsocketHelper) Run(ctx context.Context, router func(context.Context, map[string]any) error) {
	readFunc := defaultReader
	if ws.CustomReadFunc != nil {
		readFunc = *ws.CustomReadFunc
	}

	if ws.IsRunning {
		log.Warn().Msg("websocket is already running")
		return
	}
	ws.IsRunning = true
	defer func() {
		ws.IsRunning = false
	}()

	dialJob := func() error {
		return ws.Dial(ctx)
	}

	subscribeJob := func() error {
		for _, subscription := range ws.Subscriptions {
			err := ws.Write(ctx, subscription)
			if err != nil {
				log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error writing subscription to websocket")
				return err
			}
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("context cancelled, stopping websocket")
			return
		default:
			err := retrier.Retry(dialJob, 3, 1, 10)
			if err != nil {
				log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error dialing websocket")
				break
			}

			// Some providers block immediate subscription after dialing
			time.Sleep(time.Second)

			err = retrier.Retry(subscribeJob, 3, 1, 10)
			if err != nil {
				log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error subscribing to websocket")
				break
			}

			for {
				data, err := readFunc(ctx, ws.Conn)
				if err != nil {
					log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error reading from websocket")
					break
				}
				go func() {
					routerErr := router(ctx, data)
					if routerErr != nil {
						log.Warn().Err(routerErr).Str("endpoint", ws.Endpoint).Msg("error processing websocket message")
					}
				}()
			}
			ws.Close()
		}
	}
}

func (ws *WebsocketHelper) Write(ctx context.Context, message interface{}) error {
	err := wsjson.Write(ctx, ws.Conn, message)
	if err != nil {
		log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error writing to websocket")
		return err
	}
	return nil
}

func (ws *WebsocketHelper) RawWrite(ctx context.Context, message string) error {
	return ws.Conn.Write(ctx, websocket.MessageText, []byte(message))
}

func (ws *WebsocketHelper) Read(ctx context.Context, ch chan any) error {
	for {
		var t any
		err := wsjson.Read(ctx, ws.Conn, &t)
		if err != nil {
			log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error reading from websocket")
			return err
		}
		ch <- t
	}
}

func (ws *WebsocketHelper) Close() error {
	if ws.Conn == nil {
		return nil
	}
	err := ws.Conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error closing websocket")
		return err
	}
	return nil
}

func (ws *WebsocketHelper) IsAlive(ctx context.Context) error {
	if ws.Conn == nil {
		return fmt.Errorf("websocket is not running")
	}
	ctx = ws.Conn.CloseRead(ctx)

	err := ws.Conn.Ping(ctx)
	if err != nil {
		log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error pinging websocket")
		return err
	}
	return nil
}

func defaultReader(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := wsjson.Read(ctx, conn, &data)
	if err != nil {
		log.Error().Err(err).Msg("wsjson read error")
		return nil, err
	}
	return data, nil
}
