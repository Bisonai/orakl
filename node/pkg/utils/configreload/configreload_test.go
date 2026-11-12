package configreload

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type entry struct {
	Name string `json:"name"`
}

// bundleServer serves the current body (404 when status is set).
type bundleServer struct {
	mu     sync.Mutex
	body   string
	status int
}

func (b *bundleServer) set(body string, status int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.body, b.status = body, status
}

func newServer(t *testing.T, body string) *bundleServer {
	t.Helper()
	b := &bundleServer{body: body}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.status != 0 {
			w.WriteHeader(b.status)
			return
		}
		_, _ = w.Write([]byte(b.body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CONFIG_BASE_URL", srv.URL)
	t.Setenv("CONFIG_FALLBACK_URL", "")
	return b
}

type recorder struct {
	mu    sync.Mutex
	calls [][]string
	err   error
}

func (r *recorder) apply(_ context.Context, es []entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	names := make([]string, len(es))
	for i, e := range es {
		names[i] = e.Name
	}
	r.calls = append(r.calls, names)
	return nil
}

func (r *recorder) last() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		return nil
	}
	return r.calls[len(r.calls)-1]
}

func (r *recorder) n() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func newReloader(rec *recorder, count int) *Reloader[entry] {
	return New("/test_feeds.json", count, func(e entry) string { return e.Name }, rec.apply)
}

func TestReloadAddRemoveAndUnchanged(t *testing.T) {
	srv := newServer(t, `[{"name":"A"},{"name":"B"}]`)
	rec := &recorder{}
	r := newReloader(rec, 2)
	ctx := context.Background()

	applied, err := r.Reload(ctx)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, []string{"A", "B"}, rec.last())

	// same bytes: skipped
	applied, err = r.Reload(ctx)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, 1, rec.n())

	// add
	srv.set(`[{"name":"A"},{"name":"B"},{"name":"C"}]`, 0)
	applied, err = r.Reload(ctx)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, []string{"A", "B", "C"}, rec.last())

	// remove
	srv.set(`[{"name":"A"},{"name":"C"}]`, 0)
	applied, err = r.Reload(ctx)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, []string{"A", "C"}, rec.last())
}

func TestReloadRefusesBadBundles(t *testing.T) {
	for name, body := range map[string]string{
		"empty":      `[]`,
		"null":       `null`,
		"no name":    `[{"name":"A"},{"name":"B"},{"name":""},{"name":"D"}]`,
		"duplicate":  `[{"name":"A"},{"name":"B"},{"name":"A"},{"name":"D"}]`,
		"large drop": `[{"name":"A"}]`,
		"not a list": `{"name":"A"}`,
	} {
		t.Run(name, func(t *testing.T) {
			newServer(t, body)
			rec := &recorder{}
			r := newReloader(rec, 4)
			applied, err := r.Reload(context.Background())
			assert.Error(t, err)
			assert.False(t, applied)
			assert.Equal(t, 0, rec.n())
		})
	}
}

func TestReloadDropGuardAllowsHalf(t *testing.T) {
	newServer(t, `[{"name":"A"},{"name":"B"}]`)
	rec := &recorder{}
	applied, err := newReloader(rec, 4).Reload(context.Background())
	require.NoError(t, err)
	assert.True(t, applied)
}

func TestReloadFailedFetchKeepsConfig(t *testing.T) {
	srv := newServer(t, `[{"name":"A"}]`)
	rec := &recorder{}
	r := newReloader(rec, 1)
	ctx := context.Background()

	_, err := r.Reload(ctx)
	require.NoError(t, err)

	srv.set("", http.StatusNotFound)
	applied, err := r.Reload(ctx)
	assert.Error(t, err)
	assert.False(t, applied)
	assert.Equal(t, 1, rec.n())

	// non-JSON body (e.g. an error page) is a failure too
	srv.set("<html>oops</html>", 0)
	_, err = r.Reload(ctx)
	assert.Error(t, err)

	// recovers with the same content as before: nothing to apply
	srv.set(`[{"name":"A"}]`, 0)
	applied, err = r.Reload(ctx)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, 1, rec.n())
}

func TestReloadApplyErrorRetried(t *testing.T) {
	newServer(t, `[{"name":"A"}]`)
	rec := &recorder{err: assert.AnError}
	r := newReloader(rec, 1)
	ctx := context.Background()

	_, err := r.Reload(ctx)
	assert.ErrorIs(t, err, assert.AnError)

	rec.mu.Lock()
	rec.err = nil
	rec.mu.Unlock()
	applied, err := r.Reload(ctx)
	require.NoError(t, err)
	assert.True(t, applied, "failed apply must not mark the content as applied")
}

func TestRunTriggerAndInterval(t *testing.T) {
	srv := newServer(t, `[{"name":"A"}]`)
	rec := &recorder{}
	r := newReloader(rec, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trigger := make(chan os.Signal, 1)
	var done atomic.Bool
	go func() {
		r.Run(ctx, 0, trigger)
		done.Store(true)
	}()

	trigger <- os.Interrupt
	require.Eventually(t, func() bool { return rec.n() == 1 }, 2*time.Second, 10*time.Millisecond)

	cancel()
	require.Eventually(t, done.Load, time.Second, 10*time.Millisecond)

	// polling
	srv.set(`[{"name":"A"},{"name":"B"}]`, 0)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	go r.Run(ctx2, 20*time.Millisecond, nil)
	require.Eventually(t, func() bool { return len(rec.last()) == 2 }, 2*time.Second, 10*time.Millisecond)
}

func TestInterval(t *testing.T) {
	for v, want := range map[string]time.Duration{
		"":    DefaultInterval,
		"0":   0,
		"30s": 30 * time.Second,
		"bad": DefaultInterval,
		"-1m": DefaultInterval,
	} {
		t.Setenv(IntervalEnv, v)
		assert.Equal(t, want, Interval(), v)
	}
}
