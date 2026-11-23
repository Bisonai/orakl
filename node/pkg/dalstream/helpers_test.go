package dalstream

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	"nhooyr.io/websocket"
)

const testKey = "test-key"

func testConfig() Config {
	cfg := DefaultConfig()
	cfg.APIKey = testKey
	cfg.DALURL = "http://127.0.0.1:1"
	cfg.MinBackoff = 10 * time.Millisecond
	cfg.MaxBackoff = 50 * time.Millisecond
	return cfg
}

func msg(symbol, value string, aggTime int64) dalcommon.OutgoingSubmissionData {
	return dalcommon.OutgoingSubmissionData{
		Symbol:        symbol,
		Value:         value,
		AggregateTime: jsonInt(aggTime),
		Proof:         "0xproof",
		FeedHash:      "0xhash",
		Decimals:      "8",
	}
}

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

type payload struct {
	T json.Number      `json:"t"`
	D map[string][]any `json:"d"`
}

// price returns d[symbol] as (price string, aggregateTime), failing on any
// other shape.
func (p payload) price(t *testing.T, symbol string) (string, int64) {
	t.Helper()
	v, ok := p.D[symbol]
	if !ok {
		t.Fatalf("%s missing in %v", symbol, p.D)
	}
	if len(v) != 2 {
		t.Fatalf("%s: want [price, aggregateTime], got %v", symbol, v)
	}
	price, ok := v[0].(string)
	if !ok {
		t.Fatalf("%s: price %v is not a string", symbol, v[0])
	}
	n, ok := v[1].(json.Number)
	if !ok {
		t.Fatalf("%s: aggregateTime %v is not a number", symbol, v[1])
	}
	at, err := n.Int64()
	if err != nil {
		t.Fatalf("%s: aggregateTime %v: %v", symbol, n, err)
	}
	return price, at
}

type sseEvent struct {
	Event string
	Data  payload
	Raw   string
}

func parseEvent(t *testing.T, raw []byte) sseEvent {
	t.Helper()
	ev := sseEvent{Raw: string(raw)}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		switch {
		case strings.HasPrefix(line, "event: "):
			ev.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			dec := json.NewDecoder(strings.NewReader(strings.TrimPrefix(line, "data: ")))
			dec.UseNumber()
			if err := dec.Decode(&ev.Data); err != nil {
				t.Fatalf("bad data %q: %v", line, err)
			}
		}
	}
	return ev
}

// sseReader reads events (comments and retry lines are skipped) from a stream.
type sseReader struct {
	resp   *http.Response
	events chan sseEvent
	lines  chan string // every raw line, including comments
}

func openStream(t *testing.T, url string, header http.Header) (*sseReader, *http.Response) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp
	}
	r := &sseReader{resp: resp, events: make(chan sseEvent, 100), lines: make(chan string, 1000)}
	t.Cleanup(func() { resp.Body.Close() })
	go func() {
		defer close(r.events)
		sc := bufio.NewScanner(resp.Body)
		var cur strings.Builder
		for sc.Scan() {
			line := sc.Text()
			select {
			case r.lines <- line:
			default:
			}
			if line == "" {
				if s := cur.String(); strings.Contains(s, "event: ") {
					r.events <- parseEvent(t, []byte(s))
				}
				cur.Reset()
				continue
			}
			cur.WriteString(line + "\n")
		}
	}()
	return r, resp
}

func (r *sseReader) next(t *testing.T, timeout time.Duration) (sseEvent, bool) {
	t.Helper()
	select {
	case ev, ok := <-r.events:
		return ev, ok
	case <-time.After(timeout):
		return sseEvent{}, false
	}
}

// fakeDAL serves GET /symbols and /ws like DAL.
type fakeDAL struct {
	mu         sync.Mutex
	symbols    []string
	conns      []*websocket.Conn
	subscribed map[string]struct{}
	connects   int
	connected  chan *websocket.Conn
}

func newFakeDAL(symbols ...string) *fakeDAL {
	return &fakeDAL{symbols: symbols, subscribed: map[string]struct{}{}, connected: make(chan *websocket.Conn, 10)}
}

func (f *fakeDAL) setSymbols(s ...string) {
	f.mu.Lock()
	f.symbols = s
	f.mu.Unlock()
}

func (f *fakeDAL) isSubscribed(s string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.subscribed[s]
	return ok
}

func (f *fakeDAL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-API-Key") != testKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.URL.Path {
	case "/symbols":
		f.mu.Lock()
		b, _ := json.Marshal(f.symbols)
		f.mu.Unlock()
		_, _ = w.Write(b)
	case "/ws":
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		f.mu.Lock()
		f.conns = append(f.conns, c)
		f.connects++
		f.mu.Unlock()
		f.connected <- c
		for {
			_, data, err := c.Read(context.Background())
			if err != nil {
				return
			}
			var sub struct {
				Method string   `json:"method"`
				Params []string `json:"params"`
			}
			if json.Unmarshal(data, &sub) == nil && sub.Method == "SUBSCRIBE" {
				f.mu.Lock()
				for _, p := range sub.Params {
					f.subscribed[strings.TrimPrefix(p, "submission@")] = struct{}{}
				}
				f.mu.Unlock()
			}
		}
	default:
		http.NotFound(w, r)
	}
}

func send(t *testing.T, c *websocket.Conn, m dalcommon.OutgoingSubmissionData) {
	t.Helper()
	b, _ := json.Marshal(m)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("fake DAL write: %v", err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
