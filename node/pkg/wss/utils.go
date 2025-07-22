package wss

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func (ws *WebsocketHelper) Dial(ctx context.Context) error {
	dialOption := &websocket.DialOptions{}
	if ws.Proxy != "" {
		if strings.HasPrefix(ws.Endpoint, "wss") {
			ws.Endpoint = strings.Replace(ws.Endpoint, "wss", "ws", 1)
		}

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
		log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error opening websocket connection")
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

	reconnectTicker := time.NewTicker(ws.ReconnectInterval)
	inactivityTimer := time.NewTimer(ws.InactivityTimeout)
	defer reconnectTicker.Stop()
	defer inactivityTimer.Stop()

	for {
		err := ws.dialAndSubscribe(ctx)
		if err != nil {
			log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error dialing and subscribing to websocket")
			time.Sleep(time.Second)
			continue
		}
	innerLoop:
		for {
			select {
			case <-ctx.Done():
				log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
				ws.Close()
				return
			case <-reconnectTicker.C:
				log.Info().Str("endpoint", ws.Endpoint).Msg("reconnect interval exceeded during read, closing websocket")
				break innerLoop
			case <-inactivityTimer.C:
				if time.Since(ws.lastMessageTime) > ws.InactivityTimeout {
					log.Info().Str("endpoint", ws.Endpoint).Msg("inactivity timeout exceeded, closing websocket")
					break innerLoop
				}
				inactivityTimer.Reset(ws.InactivityTimeout - time.Since(ws.lastMessageTime))
			default:
				data, err := readFunc(ctx, ws.Conn)
				if err != nil {
					if isErrorNormalClosure(err) {
						break innerLoop
					}
					log.Error().Err(err).Str("endpoint", ws.Endpoint).Msg("error reading from websocket")
					break innerLoop
				}

				ws.lastMessageTime = time.Now()

				if len(data) != 0 {
					go func(context.Context, map[string]any) {
						routerErr := router(ctx, data)
						if routerErr != nil {
							log.Warn().Err(routerErr).Str("endpoint", ws.Endpoint).Msg("error processing websocket message")
						}
					}(ctx, data)
				}
			}
		}
		ws.Close()
	}
}

func (ws *WebsocketHelper) dialAndSubscribe(ctx context.Context) error {
	dialJob := func() error {
		return ws.Dial(ctx)
	}

	subscribeJob := func() error {
		for _, subscription := range ws.Subscriptions {
			switch casted := subscription.(type) {
			case []byte:
				if err := ws.RawWrite(ctx, string(casted)); err != nil {
					return err
				}
			default:
				if err := ws.Write(ctx, casted); err != nil {
					return err
				}
			}
			time.Sleep(time.Second)
		}
		return nil
	}

	err := retrier.Retry(dialJob, 3, 1, 10)
	if err != nil {
		return err
	}

	// Some providers block immediate subscription after dialing
	time.Sleep(time.Second)

	err = retrier.Retry(subscribeJob, 3, 1, 10)
	if err != nil {
		return err
	}

	return nil
}

func (ws *WebsocketHelper) Write(ctx context.Context, message interface{}) error {
	err := wsjson.Write(ctx, ws.Conn, message)
	if err != nil {
		return err
	}
	return nil
}

func (ws *WebsocketHelper) RawWrite(ctx context.Context, message string) error {
	if ws.Conn == nil {
		return errors.New("websocket is not running")
	}

	return ws.Conn.Write(ctx, websocket.MessageText, []byte(message))
}

func (ws *WebsocketHelper) Read(ctx context.Context, ch chan any) error {
	for {
		var t any
		err := wsjson.Read(ctx, ws.Conn, &t)
		if err != nil {
			log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error reading from websocket")
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
		log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error closing websocket")
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

func isErrorNormalClosure(err error) bool {
	return websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway
}

func defaultReader(ctx context.Context, conn *websocket.Conn) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := wsjson.Read(ctx, conn, &data)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	return data, nil
}
