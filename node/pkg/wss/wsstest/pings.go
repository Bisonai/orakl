package wsstest

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
)

// CountPings wraps w for a test server's websocket.Accept so that every ping
// frame the client sends on the accepted connection increments n. The
// websocket library answers pings internally and has no hook for them.
// Compression and bytes buffered before the hijack (none for a client that
// waits for the handshake reply) are not supported.
func CountPings(w http.ResponseWriter, n *atomic.Int64) http.ResponseWriter {
	return &pingCountingWriter{ResponseWriter: w, n: n}
}

type pingCountingWriter struct {
	http.ResponseWriter
	n *atomic.Int64
}

func (p *pingCountingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := p.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http.ResponseWriter does not implement http.Hijacker")
	}
	conn, brw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}
	return &pingCountingConn{Conn: conn, n: p.n}, brw, nil
}

// pingCountingConn parses the client->server frame stream as it is read.
type pingCountingConn struct {
	net.Conn
	n *atomic.Int64
	// header bytes of the frame being parsed
	hdr []byte
	// payload bytes of the current frame still to skip
	skip uint64
}

func (c *pingCountingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.feed(b[:n])
	return n, err
}

const opPing = 0x9

func (c *pingCountingConn) feed(p []byte) {
	for len(p) > 0 {
		if c.skip > 0 {
			k := min(uint64(len(p)), c.skip)
			c.skip -= k
			p = p[k:]
			continue
		}
		c.hdr = append(c.hdr, p[0])
		p = p[1:]
		if len(c.hdr) < 2 || len(c.hdr) < headerLen(c.hdr) {
			continue
		}
		if c.hdr[0]&0x0f == opPing {
			c.n.Add(1)
		}
		c.skip = payloadLen(c.hdr)
		c.hdr = c.hdr[:0]
	}
}

// headerLen returns the full header length (RFC 6455 5.2) given its first
// two bytes.
func headerLen(h []byte) int {
	l := 2
	switch h[1] & 0x7f {
	case 126:
		l += 2
	case 127:
		l += 8
	}
	if h[1]&0x80 != 0 {
		l += 4 // masking key
	}
	return l
}

func payloadLen(h []byte) uint64 {
	switch l := h[1] & 0x7f; l {
	case 126:
		return uint64(binary.BigEndian.Uint16(h[2:4]))
	case 127:
		return binary.BigEndian.Uint64(h[2:10])
	default:
		return uint64(l)
	}
}
