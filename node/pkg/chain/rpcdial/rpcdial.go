// Package rpcdial dials the kaia JSON-RPC client over an HTTP transport that
// makes a benign HTTP/2 GOAWAY (normal provider connection cycling) a
// transparent retry instead of a hard failure.
//
// Root cause: kaia's rpc httpConn.doRequest builds the POST body with
// io.NopCloser(bytes.NewReader(body)). net/http only auto-populates
// Request.GetBody for an *unwrapped* *bytes.Reader/*bytes.Buffer/*strings.Reader,
// so the wrapped body leaves GetBody nil. Without GetBody the http2 transport
// cannot replay a request whose body was already written when a GOAWAY arrives,
// producing:
//
//	http2: Transport received Server's graceful shutdown GOAWAY ... after
//	Request.Body was written; define Request.GetBody to avoid this error
//
// JSON-RPC calls are safe to replay (eth_sendRawTransaction is idempotent by tx
// hash), so we set GetBody in a RoundTripper wrapper and let the stdlib retry.
package rpcdial

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/kaiachain/kaia/networks/rpc"
)

// getBodyRoundTripper populates Request.GetBody on outgoing POSTs that lack it
// so idempotent JSON-RPC requests replay transparently on an HTTP/2 GOAWAY.
type getBodyRoundTripper struct {
	base http.RoundTripper
}

func (rt getBodyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil || req.Body == http.NoBody || req.GetBody != nil {
		return rt.base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}

	// Clone so we honor the RoundTripper contract of not mutating the caller's request.
	outReq := req.Clone(req.Context())
	outReq.Body = io.NopCloser(bytes.NewReader(body))
	outReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return rt.base.RoundTrip(outReq)
}

// newHTTPClient returns an *http.Client whose transport sets Request.GetBody.
// It wraps http.DefaultTransport (what kaia's DialHTTP uses via new(http.Client))
// so connection pooling and timeouts are unchanged on the happy path.
func newHTTPClient() *http.Client {
	return &http.Client{Transport: getBodyRoundTripper{base: http.DefaultTransport}}
}

// DialContext mirrors rpc.DialContext but, for http/https endpoints, dials over
// a client whose transport makes an HTTP/2 GOAWAY a transparent retry. Other
// schemes (ws, ipc, stdio) fall back to rpc.DialContext unchanged.
func DialContext(ctx context.Context, rawurl string) (*rpc.Client, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		return rpc.DialHTTPWithClient(rawurl, newHTTPClient())
	default:
		return rpc.DialContext(ctx, rawurl)
	}
}
