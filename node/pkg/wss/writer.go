package wss

import (
	"context"
	"errors"
	"fmt"
	"net"

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
	return w.wrap(writeJSON(ctx, w.conn, message))
}

func (w *connWriter) RawWrite(ctx context.Context, message string) error {
	if w.closed() {
		return ErrConnClosed
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
		return nil
	}
	if w.closed() || errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("%w: %w", ErrConnClosed, err)
	}
	return err
}

// writeJSON and writeText refuse an already-done ctx: nhooyr.io/websocket
// closes the whole conn when a write ctx expires, so writing with e.g. the
// cancelled ctx of a previous connection would tear down a healthy one.
func writeJSON(ctx context.Context, conn *websocket.Conn, message any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, message)
}

func writeText(ctx context.Context, conn *websocket.Conn, message string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, []byte(message))
}

// WriterFromContext returns the writer bound to the connection a router
// message arrived on. Run puts it into the ctx passed to the router. ok is
// false for contexts not derived from a router ctx.
func WriterFromContext(ctx context.Context) (ConnWriter, bool) {
	w, ok := ctx.Value(connWriterKey{}).(*connWriter)
	if !ok || w == nil {
		return nil, false
	}
	return w, true
}
