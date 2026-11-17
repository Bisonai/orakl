package wss

import (
	"context"
	"errors"
	"net"
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
	// local copy: Endpoint is read by other goroutines (logs), so Dial must not
	// rewrite the field on every reconnect
	endpoint := ws.Endpoint
	if ws.Proxy != "" {
		if strings.HasPrefix(endpoint, "wss") {
			endpoint = strings.Replace(endpoint, "wss", "ws", 1)
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
	conn, _, err := dialFunc(ctx, endpoint, dialOption)
	if err != nil {
		log.Warn().Err(err).Str("endpoint", endpoint).Msg("error opening websocket connection")
		return err
	}

	if ws.ReadLimit > 0 {
		conn.SetReadLimit(ws.ReadLimit)
	}

	// drop a conn left over from a previous Dial instead of leaking it;
	// CloseNow so Dial does not wait on a close handshake
	if old := ws.conn.Swap(conn); old != nil {
		_ = old.CloseNow()
	}
	return nil
}

// IsRunning reports whether Run is currently active.
func (ws *WebsocketHelper) IsRunning() bool {
	return ws.running.Load()
}

func (ws *WebsocketHelper) Run(ctx context.Context, router func(context.Context, map[string]any) error) {
	readFunc := defaultReader
	if ws.CustomReadFunc != nil {
		readFunc = *ws.CustomReadFunc
	}

	if !ws.running.CompareAndSwap(false, true) {
		log.Warn().Msg("websocket is already running")
		return
	}
	defer ws.running.Store(false)

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
//
// Routers run with a per-connection ctx that is cancelled when this
// connection ends and that carries a ConnWriter bound to it (see
// WriterFromContext).
func (ws *WebsocketHelper) readUntilReconnect(ctx context.Context, readFunc func(context.Context, *websocket.Conn) (map[string]interface{}, error), router func(context.Context, map[string]any) error) {
	conn := ws.conn.Load()
	if conn == nil {
		log.Warn().Str("endpoint", ws.Endpoint).Msg("websocket closed before reading, reconnecting")
		return
	}
	connCtx, cancelConn := context.WithCancel(ctx)
	defer cancelConn()
	connCtx = context.WithValue(connCtx, connWriterKey{}, &connWriter{conn: conn, done: connCtx.Done()})

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

		readCtx, cancel := context.WithDeadline(connCtx, deadline)
		data, err := readFunc(readCtx, conn)
		cancel()
		if err != nil {
			switch {
			case ctx.Err() != nil:
			case isErrorNormalClosure(err):
			// The deadline hit only if readCtx expired (cancel() already ran, so check
			// DeadlineExceeded) and the error is what nhooyr returns for it (ctx error,
			// or net.ErrClosed after its timeoutLoop closed the conn). Other errors,
			// e.g. a close frame or reset racing the deadline, keep their Warn log.
			case errors.Is(readCtx.Err(), context.DeadlineExceeded) &&
				(errors.Is(err, context.DeadlineExceeded) || errors.Is(err, net.ErrClosed)):
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

		go func(connCtx context.Context, data map[string]any) {
			routerErr := router(connCtx, data)
			if routerErr != nil {
				log.Warn().Err(routerErr).Str("endpoint", ws.Endpoint).Msg("error processing websocket message")
			}
		}(connCtx, data)
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

// Write sends message as JSON on the current connection. It returns
// ErrNotConnected when there is none (before the first dial, after Close or
// between reconnects).
func (ws *WebsocketHelper) Write(ctx context.Context, message interface{}) error {
	conn := ws.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}

	return writeJSON(ctx, conn, message)
}

// RawWrite sends message as a text frame on the current connection. It
// returns ErrNotConnected when there is none.
func (ws *WebsocketHelper) RawWrite(ctx context.Context, message string) error {
	conn := ws.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}

	return writeText(ctx, conn, message)
}

func (ws *WebsocketHelper) Read(ctx context.Context, ch chan any) error {
	conn := ws.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}
	for {
		var t any
		err := wsjson.Read(ctx, conn, &t)
		if err != nil {
			log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error reading from websocket")
			return err
		}
		ch <- t
	}
}

// Close detaches the current connection and closes it. It is idempotent and
// safe to call concurrently with writers; later writes get ErrNotConnected.
func (ws *WebsocketHelper) Close() error {
	conn := ws.conn.Swap(nil)
	if conn == nil {
		return nil
	}
	err := conn.Close(websocket.StatusNormalClosure, "")
	if err != nil {
		log.Warn().Err(err).Str("endpoint", ws.Endpoint).Msg("error closing websocket")
		return err
	}
	return nil
}

func (ws *WebsocketHelper) IsAlive(ctx context.Context) error {
	conn := ws.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}
	ctx = conn.CloseRead(ctx)

	err := conn.Ping(ctx)
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
