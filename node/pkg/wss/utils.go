package wss

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/utils/retrier"
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
	if !ws.running.CompareAndSwap(false, true) {
		log.Warn().Str("endpoint", ws.Endpoint).Msg("websocket is already running")
		return
	}
	defer ws.running.Store(false)

	readFunc := defaultReader
	if ws.CustomReadFunc != nil {
		readFunc = *ws.CustomReadFunc
	}

	for {
		if ctx.Err() != nil {
			log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
			return
		}

		session, err := ws.dialAndSubscribe(ctx)
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

		ws.readUntilReconnect(ctx, session, readFunc, router)

		if ctx.Err() != nil {
			log.Info().Str("endpoint", ws.Endpoint).Msg("context cancelled, stopping websocket")
			return
		}
	}
}

// connSession is one dialed connection with its per-connection ctx and the
// keepalive running on it.
type connSession struct {
	conn *websocket.Conn
	// ctx is cancelled when the connection ends and carries the ConnWriter
	// bound to it (see WriterFromContext)
	ctx           context.Context
	cancel        context.CancelCauseFunc
	keepaliveDone chan struct{}
	writer        *connWriter
}

// startSession sets up the per-connection ctx and writer for conn and starts
// the keepalive (WithKeepalive), if any, right away, so it also runs while
// subscribing.
func (ws *WebsocketHelper) startSession(ctx context.Context, conn *websocket.Conn) *connSession {
	connCtx, cancel := context.WithCancelCause(ctx)
	writer := &connWriter{conn: conn, ctx: connCtx}
	s := &connSession{
		conn:          conn,
		ctx:           ContextWithWriter(connCtx, writer),
		cancel:        cancel,
		keepaliveDone: make(chan struct{}),
		writer:        writer,
	}
	if ws.keepaliveFn == nil {
		close(s.keepaliveDone)
		return s
	}
	// read after Dial, so a custom dial func can set it for this conn
	interval := ws.keepaliveInterval()
	go func() {
		defer close(s.keepaliveDone)
		ws.runKeepalive(s.ctx, interval, writer, func() { cancel(errKeepaliveFailed) })
	}()
	return s
}

// endSession ends the connection: cancels its ctx (aborting in-flight
// keepalive / router writes), closes the conn and waits for the keepalive to
// stop.
func (ws *WebsocketHelper) endSession(s *connSession) {
	s.cancel(nil)
	// close the conn of this session, not whatever is current by then
	ws.closeConn(s.conn)
	<-s.keepaliveDone
}

// readUntilReconnect reads from the session's connection until it should be
// replaced: read error, inactivity timeout or reconnect interval. Each read is
// bounded by a deadline, so a connection that is open but silent cannot block
// forever. nhooyr.io/websocket closes the conn when a read ctx expires, so
// closing it here afterwards does not block.
//
// Routers and the read func run with the session ctx, which is cancelled when
// this connection ends and carries a ConnWriter bound to it. The session is
// ended before this returns.
func (ws *WebsocketHelper) readUntilReconnect(ctx context.Context, session *connSession, readFunc func(context.Context, *websocket.Conn) (map[string]interface{}, error), router func(context.Context, map[string]any) error) {
	defer ws.endSession(session)
	conn := session.conn
	connCtx := session.ctx
	session.writer.reading.Store(true)

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
			case errors.Is(context.Cause(connCtx), errKeepaliveFailed):
				log.Warn().Str("endpoint", ws.Endpoint).Int("failures", MaxKeepaliveFailures).Msg("websocket keepalive failing, reconnecting")
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

// subscribeWait is the pause after dialing and after each subscription.
func (ws *WebsocketHelper) subscribeWait() time.Duration {
	if ws.subscribeDelay > 0 {
		return ws.subscribeDelay
	}
	return time.Second
}

// dialAndSubscribe dials, starts the connection's session (incl. keepalive)
// and subscribes. On a subscribe error the session is ended before returning.
func (ws *WebsocketHelper) dialAndSubscribe(ctx context.Context) (*connSession, error) {
	dialJob := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return ws.Dial(ctx)
	}

	err := retrier.Retry(dialJob, 3, 1, 10)
	if err != nil {
		return nil, err
	}
	conn := ws.conn.Load()
	if conn == nil {
		return nil, ErrNotConnected
	}
	session := ws.startSession(ctx, conn)
	// Subscribe on the session ctx: it follows the run ctx (shutdown is not
	// held up by a blocked write; aborting one closes this half-subscribed
	// conn, which is dropped on error anyway) and ends when the keepalive
	// gives up on the conn.
	subCtx := session.ctx

	subscribeJob := func() error {
		if err := subCtx.Err(); err != nil {
			return err
		}
		for _, subscription := range ws.Subscriptions {
			switch casted := subscription.(type) {
			case []byte:
				if err := writeText(subCtx, conn, string(casted), true); err != nil {
					return err
				}
			default:
				if err := writeJSON(subCtx, conn, casted, true); err != nil {
					return err
				}
			}
			if err := sleepCtx(subCtx, ws.subscribeWait()); err != nil {
				return err
			}
		}
		return nil
	}

	// Some providers block immediate subscription after dialing
	err = sleepCtx(subCtx, ws.subscribeWait())
	if err == nil {
		err = retrier.Retry(subscribeJob, 3, 1, 10)
	}
	if err != nil {
		if cause := context.Cause(subCtx); errors.Is(cause, errKeepaliveFailed) {
			err = cause
		}
		// end the half-open conn and its keepalive so neither leaks on retry or exit
		ws.endSession(session)
		return nil, err
	}

	return session, nil
}

// Write sends message as JSON on the current connection. It returns
// ErrNotConnected when there is none (before the first dial, after Close or
// between reconnects).
func (ws *WebsocketHelper) Write(ctx context.Context, message interface{}) error {
	conn := ws.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}

	return writeJSON(ctx, conn, message, false)
}

// RawWrite sends message as a text frame on the current connection. It
// returns ErrNotConnected when there is none.
func (ws *WebsocketHelper) RawWrite(ctx context.Context, message string) error {
	conn := ws.conn.Load()
	if conn == nil {
		return ErrNotConnected
	}

	return writeText(ctx, conn, message, false)
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
	return ws.closeAndLog(conn)
}

// closeConn detaches conn if it is still current, then closes it.
func (ws *WebsocketHelper) closeConn(conn *websocket.Conn) {
	ws.conn.CompareAndSwap(conn, nil)
	_ = ws.closeAndLog(conn)
}

func (ws *WebsocketHelper) closeAndLog(conn *websocket.Conn) error {
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
