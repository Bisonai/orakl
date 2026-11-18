//nolint:all
package orangex

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/wss/wsstest"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

// logs captures everything logged by the package under test.
var logs syncBuffer

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestMain(m *testing.M) {
	// set before any test goroutine logs
	log.Logger = log.Output(io.MultiWriter(os.Stderr, &logs))
	os.Exit(m.Run())
}

type fakeConn struct {
	accepted time.Time
	pings    atomic.Int64
	mu       sync.Mutex
	frames   []string
}

func (c *fakeConn) snapshotFrames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.frames...)
}

// fakeServer is an orangex stand-in: it records each connection's pings and
// text frames and, on the first text frame (the subscribe), calls
// onSubscribe, which may push messages to the client.
type fakeServer struct {
	*httptest.Server
	mu    sync.Mutex
	conns []*fakeConn
}

func newFakeServer(t *testing.T, onSubscribe func(ctx context.Context, conn *websocket.Conn)) *fakeServer {
	t.Helper()
	fs := &fakeServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fc := &fakeConn{accepted: time.Now()}
		conn, err := websocket.Accept(wsstest.CountPings(w, &fc.pings), r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		fs.mu.Lock()
		fs.conns = append(fs.conns, fc)
		fs.mu.Unlock()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		for {
			// reading also answers the client's pings
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			fc.mu.Lock()
			fc.frames = append(fc.frames, string(data))
			first := len(fc.frames) == 1
			fc.mu.Unlock()
			if first && onSubscribe != nil {
				go onSubscribe(ctx, conn)
			}
		}
	}))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *fakeServer) url() string { return "ws" + fs.URL[len("http"):] }

func (fs *fakeServer) snapshot() []*fakeConn {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return append([]*fakeConn(nil), fs.conns...)
}

func (fs *fakeServer) waitConns(n int, timeout time.Duration) []*fakeConn {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := fs.snapshot(); len(s) >= n {
			return s
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fs.snapshot()
}

func testFetcher(t *testing.T, fs *fakeServer, feeds []string, pingInterval, readTimeout time.Duration) (*OrangeXFetcher, chan *common.FeedData) {
	t.Helper()
	separated := map[string][]int32{}
	for i, feed := range feeds {
		separated[feed] = []int32{int32(i)}
	}
	buffer := make(chan *common.FeedData, 1000)
	f, err := newFetcher(context.Background(),
		connParams{endpoint: fs.url(), pingInterval: pingInterval, readTimeout: readTimeout},
		common.WithFeedMaps(common.FeedMaps{Separated: separated}),
		common.WithFeedDataBuffer(buffer))
	require.NoError(t, err)
	return f, buffer
}

// runFetcher runs f in the background; the returned stop cancels it and fails
// the test unless Run returns promptly.
func runFetcher(t *testing.T, f *OrangeXFetcher) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		f.Run(ctx)
		close(done)
	}()
	stop := func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("Run did not return promptly after context cancel")
		}
	}
	t.Cleanup(stop)
	return stop
}

func tickerMessage(instrument string) string {
	return `{"params":{"data":{"timestamp":"1791866634698","stats":{"volume":"12.5"},"last_price":"65000.5","instrument_name":"` + instrument + `"},"channel":"ticker.` + instrument + `.raw"},"method":"subscription","jsonrpc":"2.0"}`
}

func send(ctx context.Context, conn *websocket.Conn, msg string) error {
	return conn.Write(ctx, websocket.MessageText, []byte(msg))
}

// A connection that is pinged but gets no data reconnects after readTimeout,
// and every connection, old and new, is pinged.
func TestRun_SilentConnPingsAndReconnects(t *testing.T) {
	fs := newFakeServer(t, nil)
	const readTimeout = 400 * time.Millisecond
	f, _ := testFetcher(t, fs, []string{"BTC-USDT"}, 50*time.Millisecond, readTimeout)
	stop := runFetcher(t, f)

	conns := fs.waitConns(3, 15*time.Second)
	require.GreaterOrEqual(t, len(conns), 3, "silent conn did not reconnect")
	require.Eventually(t, func() bool { return conns[2].pings.Load() > 0 }, 2*time.Second, 10*time.Millisecond, "no pings after reconnect")
	stop()

	want, err := json.Marshal(BuildSubscriptions([]string{"BTC-USDT"})[0])
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		// each conn lives ~1s pre-subscribe wait + 1s post-subscribe wait +
		// readTimeout, pinged every 50ms
		assert.GreaterOrEqual(t, conns[i].pings.Load(), int64(15), "conn %d pings", i)
		assert.Equal(t, []string{string(want)}, conns[i].snapshotFrames(), "conn %d frames", i)

		gap := conns[i+1].accepted.Sub(conns[i].accepted)
		assert.GreaterOrEqual(t, gap, 2*time.Second+readTimeout-200*time.Millisecond, "conn %d ended before readTimeout", i)
		assert.LessOrEqual(t, gap, 2*time.Second+readTimeout+2*time.Second, "conn %d not ended by readTimeout", i)
	}
}

// Ticker data counts as activity: a streaming conn stays open past readTimeout.
func TestRun_DataKeepsConnOpen(t *testing.T) {
	fs := newFakeServer(t, func(ctx context.Context, conn *websocket.Conn) {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if send(ctx, conn, tickerMessage("BTC-USDT")) != nil {
					return
				}
			}
		}
	})
	const readTimeout = 300 * time.Millisecond
	f, buffer := testFetcher(t, fs, []string{"BTC-USDT"}, 50*time.Millisecond, readTimeout)
	stop := runFetcher(t, f)

	require.Eventually(t, func() bool { return len(buffer) > 0 }, 5*time.Second, 10*time.Millisecond, "no ticker reached the buffer")
	// several readTimeouts of streaming
	time.Sleep(5 * readTimeout)
	stop()

	assert.Len(t, fs.snapshot(), 1, "streaming conn reconnected")
	got := <-buffer
	assert.Equal(t, int32(0), got.FeedID)
	assert.Equal(t, 65000.5, got.Value)
}

func TestRun_OneConnectionPerChunk(t *testing.T) {
	fs := newFakeServer(t, nil)
	feeds := feedNames(61)
	f, _ := testFetcher(t, fs, feeds, 50*time.Millisecond, time.Minute)
	stop := runFetcher(t, f)

	wantChunks := BuildSubscriptions(feeds)
	require.Len(t, wantChunks, 3)
	require.Eventually(t, func() bool {
		conns := fs.snapshot()
		if len(conns) < len(wantChunks) {
			return false
		}
		for _, c := range conns {
			if len(c.snapshotFrames()) == 0 || c.pings.Load() < 3 {
				return false
			}
		}
		return true
	}, 10*time.Second, 10*time.Millisecond, "not every chunk connected, subscribed and got pings")
	stop()

	conns := fs.snapshot()
	require.Len(t, conns, len(wantChunks), "one connection per chunk")
	var got, want [][]string
	for _, c := range conns {
		frames := c.snapshotFrames()
		require.Len(t, frames, 1, "one subscribe per connection")
		var sub Subscription
		require.NoError(t, json.Unmarshal([]byte(frames[0]), &sub))
		got = append(got, sub.Params.Channels)
	}
	for _, sub := range wantChunks {
		want = append(want, sub.Params.Channels)
	}
	// conns dial concurrently, so compare as sets of chunks
	byFirst := func(c [][]string) {
		sort.Slice(c, func(i, j int) bool { return c[i][0] < c[j][0] })
	}
	byFirst(got)
	byFirst(want)
	assert.Equal(t, want, got)
}

// Error replies are logged (9901 at warn, others at error) and the conn keeps
// delivering tickers.
func TestRun_ErrorReplyLogged(t *testing.T) {
	const limitReply = `{"id":"1","jsonrpc":"2.0","error":{"code":1005,"message":"Subscription channel limit exceeded","data":{"limit":"30","requestedNew":"30","projected":"60","current":"30","marker":"run-test"}}}`
	const autoClose = `{"id":"-1","jsonrpc":"2.0","error":{"code":9901,"message":"websocket closing connection automatically run-test"}}`
	fs := newFakeServer(t, func(ctx context.Context, conn *websocket.Conn) {
		for _, msg := range []string{limitReply, autoClose, tickerMessage("BTC-USDT")} {
			if send(ctx, conn, msg) != nil {
				return
			}
		}
	})
	f, buffer := testFetcher(t, fs, []string{"BTC-USDT"}, 50*time.Millisecond, time.Minute)
	stop := runFetcher(t, f)

	var limitLine, autoCloseLine string
	require.Eventually(t, func() bool {
		limitLine, autoCloseLine = "", ""
		for _, line := range strings.Split(logs.String(), "\n") {
			if !strings.Contains(line, "orangex returned an error response") {
				continue
			}
			if strings.Contains(line, `"marker":"run-test"`) {
				limitLine = line
			}
			if strings.Contains(line, "automatically run-test") {
				autoCloseLine = line
			}
		}
		return limitLine != "" && autoCloseLine != "" && len(buffer) > 0
	}, 5*time.Second, 10*time.Millisecond, "error replies not logged or ticker not delivered")
	stop()

	assert.Contains(t, limitLine, `"level":"error"`)
	assert.Contains(t, limitLine, `"code":1005`)
	assert.Contains(t, autoCloseLine, `"level":"warn"`)
	assert.Contains(t, autoCloseLine, `"code":9901`)
	assert.Len(t, fs.snapshot(), 1, "error reply caused a reconnect")
}
