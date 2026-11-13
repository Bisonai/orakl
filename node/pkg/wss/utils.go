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

	for {
		if ctx.Err() != nil {
			log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
			return
		}

		err := ws.dialAndSubscribe(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
				return
			}
			log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error dialing and subscribing to websocket")
			if sleepCtx(ctx, time.Second) != nil {
				log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
				return
			}
			continue
		}

		ws.readUntilReconnect(ctx, readFunc, router)
		ws.Close()

		if ctx.Err() != nil {
			log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
			return
		}
	}
}

// readUntilReconnect reads from the current connection until it should be
// replaced: read error, inactivity timeout or reconnect interval. Each read is
// bounded by a deadline, so a connection that is open but silent cannot block
// forever. nhooyr.io/websocket closes the conn when a read ctx expires; the
// caller's Close afterwards returns nil without blocking.
func (ws *WebsocketHelper) readUntilReconnect(ctx context.Context, readFunc func(context.Context, *websocket.Conn) (map[string]interface{}, error), router func(context.Context, map[string]any) error) {
	// both windows start once dial + subscribe are done
	reconnectAt := time.Now().Add(ws.ReconnectInterval)
	lastMessageTime := time.Now()

	for {
		deadline := lastMessageTime.Add(ws.InactivityTimeout)
		reason := "inactivity timeout exceeded, closing websocket"
		if reconnectAt.Before(deadline) {
			deadline = reconnectAt
			reason = "reconnect interval exceeded, closing websocket"
		}

		readCtx, cancel := context.WithDeadline(ctx, deadline)
		data, err := readFunc(readCtx, ws.Conn)
		cancel()
		if err != nil {
			switch {
			case ctx.Err() != nil:
			case isErrorNormalClosure(err):
			// a close frame from the server wins over the deadline, so its reason is logged;
			// cancel() already ran, so only DeadlineExceeded means the deadline hit
			case websocket.CloseStatus(err) == -1 && errors.Is(readCtx.Err(), context.DeadlineExceeded):
				log.Info().Str("endpoint", ws.Endpoint).Msg(reason)
			default:
				log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error reading from websocket, reconnecting")
			}
			return
		}

		// Only non-empty data counts as activity. Readers return (nil, nil) for
		// app-level heartbeats (e.g. "pong"), which prove the socket is up but not
		// that the subscription delivers; counting them would keep a dead
		// subscription open until the reconnect interval.
		if len(data) == 0 {
			continue
		}
		lastMessageTime = time.Now()

		go func(context.Context, map[string]any) {
			routerErr := router(ctx, data)
			if routerErr != nil {
				log.Warn().Err(routerErr).Str("endpoint", ws.Endpoint).Msg("error processing websocket message")
			}
		}(ctx, data)
	}
}

func (ws *WebsocketHelper) dialAndSubscribe(ctx context.Context) error {
	dialJob := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return ws.Dial(ctx)
	}

	subscribeJob := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
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
			if err := sleepCtx(ctx, time.Second); err != nil {
				return err
			}
		}
		return nil
	}

	err := retrier.Retry(dialJob, 3, 1, 10)
	if err != nil {
		return err
	}

	// Some providers block immediate subscription after dialing
	err = sleepCtx(ctx, time.Second)
	if err == nil {
		err = retrier.Retry(subscribeJob, 3, 1, 10)
	}
	if err != nil {
		// close half-open conn so it isn't leaked on retry or exit
		ws.Close()
		return err
	}

	return nil
}

func (ws *WebsocketHelper) Write(ctx context.Context, message interface{}) error {
	if ws.Conn == nil {
		return errors.New("websocket is not running")
	}

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

// sleepCtx sleeps for d or until ctx is done, returning ctx.Err() in the latter case
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isErrorNormalClosure(err error) bool {
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway || status == websocket.StatusNoStatusRcvd
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
