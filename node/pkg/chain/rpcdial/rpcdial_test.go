package rpcdial

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

// captureRoundTripper records the request it is asked to send and returns a
// canned 200 response so RoundTrip can be exercised without a network.
type captureRoundTripper struct {
	got *http.Request
}

func (c *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	c.got = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Header:     make(http.Header),
	}, nil
}

// newKaiaStyleRequest mirrors how kaia's rpc httpConn.doRequest builds the POST:
// an io.NopCloser wrapping a bytes.Reader, which leaves GetBody nil.
func newKaiaStyleRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://example.com", io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.ContentLength = int64(len(body))
	if req.GetBody != nil {
		t.Fatal("precondition failed: kaia-style request already has GetBody set")
	}
	return req
}

func TestGetBodyRoundTripperSetsGetBody(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["0xdeadbeef"],"id":1}`)
	base := &captureRoundTripper{}
	rt := getBodyRoundTripper{base: base}

	resp, err := rt.RoundTrip(newKaiaStyleRequest(t, body))
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	if base.got.GetBody == nil {
		t.Fatal("GetBody not set on forwarded request; http2 GOAWAY replay impossible")
	}

	// GetBody must yield the full body, repeatably (the http2 transport calls it on retry).
	for i := 0; i < 2; i++ {
		r, err := base.got.GetBody()
		if err != nil {
			t.Fatalf("GetBody() call %d: %v", i, err)
		}
		got, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatalf("read GetBody %d: %v", i, err)
		}
		if !bytes.Equal(got, body) {
			t.Fatalf("GetBody %d = %q, want %q", i, got, body)
		}
	}

	// The forwarded body itself must still be readable and intact.
	sent, err := io.ReadAll(base.got.Body)
	if err != nil {
		t.Fatalf("read forwarded body: %v", err)
	}
	if !bytes.Equal(sent, body) {
		t.Fatalf("forwarded body = %q, want %q", sent, body)
	}
}

func TestGetBodyRoundTripperPreservesExistingGetBody(t *testing.T) {
	body := []byte(`{"id":1}`)
	base := &captureRoundTripper{}
	rt := getBodyRoundTripper{base: base}

	// A request built directly from a *bytes.Reader already has GetBody (net/http sets it).
	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://example.com", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if req.GetBody == nil {
		t.Fatal("precondition failed: expected net/http to set GetBody")
	}
	sentinel := req.GetBody

	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	// Happy path untouched: same request forwarded, GetBody left as-is.
	if base.got != req {
		t.Fatal("request with existing GetBody should be forwarded unmodified")
	}
	if _, err := sentinel(); err != nil {
		t.Fatalf("original GetBody broken: %v", err)
	}
}

func TestNewHTTPClientWiresGetBodyTransport(t *testing.T) {
	// Guards the regression the PR fixes: the http client must carry the
	// GetBody-setting transport, wrapping the shared default transport so the
	// happy path (pooling/timeouts) is unchanged.
	rt, ok := newHTTPClient().Transport.(getBodyRoundTripper)
	if !ok {
		t.Fatalf("transport = %T, want getBodyRoundTripper", newHTTPClient().Transport)
	}
	if rt.base != http.DefaultTransport {
		t.Fatal("getBodyRoundTripper base is not http.DefaultTransport")
	}
}

func TestDialContextHTTP(t *testing.T) {
	// DialHTTPWithClient is lazy (no network), so this just checks construction.
	c, err := DialContext(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	if c == nil {
		t.Fatal("nil rpc client")
	}
	c.Close()
}
