package wss

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	// ErrNotConnected is returned by WebsocketHelper writes when there is no
	// live connection.
	ErrNotConnected = errors.New("websocket is not connected")
	// ErrConnClosed is returned by a connection-bound ConnWriter once the
	// helper has ended its connection. A write that fails before the helper
	// notices the conn is gone (e.g. broken pipe) returns the underlying error.
	ErrConnClosed = errors.New("websocket connection closed")
	// ErrNoConnWriter is returned by code that needs WriterFromContext when
	// the ctx carries no writer.
	ErrNoConnWriter = errors.New("no websocket writer in context")
)

// ConnWriter writes to a websocket connection.
//
// *WebsocketHelper implements it against whatever connection is current. The
// writer returned by WriterFromContext is bound to one connection instead.
type ConnWriter interface {
	Write(ctx context.Context, message any) error
	RawWrite(ctx context.Context, message string) error
}

var _ ConnWriter = (*WebsocketHelper)(nil)

type connWriterKey struct{}

// connWriter is bound to a single connection. Once that connection has ended
// (done is closed) writes fail with ErrConnClosed; they never reach a newer
// connection.
type connWriter struct {
	conn *websocket.Conn
	done <-chan struct{}
}

func (w *connWriter) Write(ctx context.Context, message any) error {
	if w.closed() {
		return ErrConnClosed
	}
	return w.wrap(ctx, writeJSON(ctx, w.conn, message))
}

func (w *connWriter) RawWrite(ctx context.Context, message string) error {
	if w.closed() {
		return ErrConnClosed
	}
	return w.wrap(ctx, writeText(ctx, w.conn, message))
}

func (w *connWriter) closed() bool {
	select {
	case <-w.done:
		return true
	default:
		return false
	}
}

// wrap marks errors caused by the connection having gone away with
// ErrConnClosed, keeping the underlying error in the message. The caller's own
// ctx errors are returned as is.
func (w *connWriter) wrap(ctx context.Context, err error) error {
	if err == nil || ctx.Err() != nil {
		return err
	}
	if w.closed() || errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("%w: %w", ErrConnClosed, err)
	}
	return err
}

// WriteTimeout bounds every write made through the helper or a ConnWriter.
const WriteTimeout = 10 * time.Second

// writeCtx returns the ctx a write actually runs with. nhooyr.io/websocket
// closes the whole conn when a write ctx ends mid-write, so a caller ctx (e.g.
// a router ctx) must not be able to tear down a healthy conn: the write runs
// detached from the caller's cancellation and deadline, bounded by
// WriteTimeout instead. An already-done caller ctx is refused up front.
func writeCtx(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), WriteTimeout)
	return wctx, cancel, nil
}

func writeJSON(ctx context.Context, conn *websocket.Conn, message any) error {
	wctx, cancel, err := writeCtx(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	return wsjson.Write(wctx, conn, message)
}

func writeText(ctx context.Context, conn *websocket.Conn, message string) error {
	wctx, cancel, err := writeCtx(ctx)
	if err != nil {
		return err
	}
	defer cancel()
	return conn.Write(wctx, websocket.MessageText, []byte(message))
}

// WriterFromContext returns the writer bound to the connection a router
// message arrived on. Run puts it into the ctx passed to the router and to the
// read func. ok is false for contexts not derived from such a ctx.
func WriterFromContext(ctx context.Context) (ConnWriter, bool) {
	w, ok := ctx.Value(connWriterKey{}).(ConnWriter)
	return w, ok
}

// ContextWithWriter returns a copy of ctx carrying w, as Run does for each
// connection. Useful to test routers that reply via WriterFromContext.
func ContextWithWriter(ctx context.Context, w ConnWriter) context.Context {
	return context.WithValue(ctx, connWriterKey{}, w)
}
