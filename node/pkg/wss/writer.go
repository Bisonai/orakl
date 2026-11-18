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
	// the caller's own ctx error is returned as is
	if err := ctx.Err(); err != nil {
		return err
	}
	return w.wrap(writeJSON(ctx, w.conn, message))
}

func (w *connWriter) RawWrite(ctx context.Context, message string) error {
	if w.closed() {
		return ErrConnClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return w.wrap(writeText(ctx, w.conn, message))
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
// ErrConnClosed, keeping the underlying error in the message.
func (w *connWriter) wrap(err error) error {
	if err == nil {
		return err
	}
	if w.closed() || errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("%w: %w", ErrConnClosed, err)
	}
	return err
}

// WriteTimeout bounds every write made through the helper or a ConnWriter.
const WriteTimeout = 10 * time.Second

// boundCtx returns the ctx a write actually runs with, bounded by
// WriteTimeout. nhooyr.io/websocket closes the whole conn when a write ctx
// ends mid-write, so unless attached is set, the write runs detached from the
// caller's cancellation and deadline: a caller ctx (e.g. a router ctx) must
// not be able to tear down a healthy conn. An already-done caller ctx is
// refused up front either way.
func boundCtx(ctx context.Context, attached bool) (context.Context, context.CancelFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if !attached {
		ctx = context.WithoutCancel(ctx)
	}
	wctx, cancel := context.WithTimeout(ctx, WriteTimeout)
	return wctx, cancel, nil
}

func writeJSON(ctx context.Context, conn *websocket.Conn, message any) error {
	return writeJSONCtx(ctx, conn, message, false)
}

func writeText(ctx context.Context, conn *websocket.Conn, message string) error {
	return writeTextCtx(ctx, conn, message, false)
}

func writeJSONCtx(ctx context.Context, conn *websocket.Conn, message any, attached bool) error {
	wctx, cancel, err := boundCtx(ctx, attached)
	if err != nil {
		return err
	}
	defer cancel()
	return wsjson.Write(wctx, conn, message)
}

func writeTextCtx(ctx context.Context, conn *websocket.Conn, message string, attached bool) error {
	wctx, cancel, err := boundCtx(ctx, attached)
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
