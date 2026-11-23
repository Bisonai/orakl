package apiv2

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setHistoryCacheVars(t *testing.T, ttl time.Duration, maxEntries int) {
	t.Helper()
	oldTTL, oldMax := historyCacheTTL, historyCacheMaxEntries
	historyCacheTTL, historyCacheMaxEntries = ttl, maxEntries
	t.Cleanup(func() { historyCacheTTL, historyCacheMaxEntries = oldTTL, oldMax })
}

func TestHistoryServedFromCache(t *testing.T) {
	srv, h := newHistoryServer(t, true)
	now := time.Now()
	h.Record(testSymbol, 1, "6", now.Add(-time.Hour))
	require.NoError(t, h.Flush(context.Background()))

	url := srv.URL + "/history/" + testSymbol + "," + otherSymbol
	status, first := getHistory(t, url, namedKey)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, first[testSymbol].Points, 1)

	// a newer sample is not visible until the cached response expires
	h.Record(testSymbol, 2, "6", now)
	require.NoError(t, h.Flush(context.Background()))
	status, again := getHistory(t, url, namedKey)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, first, again)

	// symbol order doesn't matter
	status, reordered := getHistory(t, srv.URL+"/history/"+otherSymbol+","+testSymbol, namedKey)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, first, reordered)
}

func TestHistoryCacheExpires(t *testing.T) {
	setHistoryCacheVars(t, 0, historyCacheMaxEntries)
	srv, h := newHistoryServer(t, true)
	now := time.Now()
	h.Record(testSymbol, 1, "6", now.Add(-time.Hour))
	require.NoError(t, h.Flush(context.Background()))

	url := srv.URL + "/history/" + testSymbol
	_, first := getHistory(t, url, namedKey)
	require.Len(t, first[testSymbol].Points, 1)

	h.Record(testSymbol, 2, "6", now)
	require.NoError(t, h.Flush(context.Background()))
	_, second := getHistory(t, url, namedKey)
	assert.Len(t, second[testSymbol].Points, 2)
}

func TestHistoryCacheSharesInFlightBuild(t *testing.T) {
	c := newHistoryCache()
	var builds atomic.Int32
	release := make(chan struct{})
	build := func(context.Context) ([]byte, error) {
		builds.Add(1)
		<-release
		return []byte("ok"), nil
	}

	var wg sync.WaitGroup
	results := make([][]byte, 5)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body, err := c.get(context.Background(), "k", build)
			assert.NoError(t, err)
			results[i] = body
		}(i)
	}
	require.Eventually(t, func() bool { return builds.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond) // let the other callers reach the wait
	close(release)
	wg.Wait()

	assert.Equal(t, int32(1), builds.Load())
	for _, body := range results {
		assert.Equal(t, []byte("ok"), body)
	}
}

func TestHistoryCacheDoesNotCacheErrors(t *testing.T) {
	c := newHistoryCache()
	calls := 0
	build := func(context.Context) ([]byte, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("boom")
		}
		return []byte("ok"), nil
	}
	_, err := c.get(context.Background(), "k", build)
	require.Error(t, err)
	body, err := c.get(context.Background(), "k", build)
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), body)
	_, _ = c.get(context.Background(), "k", build)
	assert.Equal(t, 2, calls, "success is cached")
}

func TestHistoryCacheBounded(t *testing.T) {
	setHistoryCacheVars(t, time.Minute, 1)
	c := newHistoryCache()
	calls := map[string]int{}
	build := func(key string) func(context.Context) ([]byte, error) {
		return func(context.Context) ([]byte, error) {
			calls[key]++
			return []byte(key), nil
		}
	}
	for range 2 {
		_, _ = c.get(context.Background(), "a", build("a"))
		_, _ = c.get(context.Background(), "b", build("b"))
	}
	assert.Equal(t, 1, calls["a"], "first key cached")
	assert.Equal(t, 2, calls["b"], "full cache serves without storing")
	assert.Len(t, c.entries, 1)
}

func TestHistoryCacheBuildSurvivesCallerCancel(t *testing.T) {
	c := newHistoryCache()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	body, err := c.get(ctx, "k", func(buildCtx context.Context) ([]byte, error) {
		return []byte("ok"), buildCtx.Err()
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), body)
}

func TestHistoryCacheKey(t *testing.T) {
	assert.Equal(t,
		historyCacheKey(false, []string{"B-C", "A-B"}, 24*time.Hour, time.Hour),
		historyCacheKey(false, []string{"A-B", "B-C"}, 24*time.Hour, time.Hour))
	assert.NotEqual(t,
		historyCacheKey(false, []string{"A-B"}, 24*time.Hour, time.Hour),
		historyCacheKey(false, []string{"A-B"}, 24*time.Hour, 5*time.Minute))
	assert.NotEqual(t,
		historyCacheKey(true, []string{"A-B"}, 24*time.Hour, time.Hour),
		historyCacheKey(false, []string{"A-B"}, 24*time.Hour, time.Hour))
}
