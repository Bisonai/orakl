package apiv2

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// historyCacheTTL is how long a built /history response is served from memory.
// History changes only per flush and window, so a short TTL bounds SQLite to
// about one query per distinct request per TTL. Vars so tests can change them.
var (
	historyCacheTTL        = time.Minute
	historyCacheMaxEntries = 128
	historyBuildTimeout    = 10 * time.Second
)

type historyCacheEntry struct {
	body    []byte
	err     error
	expires time.Time
	done    chan struct{} // closed once body/err are set
}

// historyCache keeps encoded /history responses for historyCacheTTL and lets
// concurrent requests for the same key share one build.
type historyCache struct {
	mu      sync.Mutex
	entries map[string]*historyCacheEntry
}

func newHistoryCache() *historyCache {
	return &historyCache{entries: make(map[string]*historyCacheEntry)}
}

// historyCacheKey identifies a request; symbol order doesn't change the response.
func historyCacheKey(all bool, symbols []string, window, step time.Duration) string {
	sel := "all"
	if !all {
		sorted := append([]string(nil), symbols...)
		sort.Strings(sorted)
		sel = strings.Join(sorted, ",")
	}
	return window.String() + "|" + step.String() + "|" + sel
}

// get returns the cached body for key, waiting for an in-flight build, or
// builds it. Errors are not cached. The build runs detached from the caller's
// cancellation so one disconnecting client doesn't fail the others waiting on it.
func (c *historyCache) get(ctx context.Context, key string, build func(context.Context) ([]byte, error)) ([]byte, error) {
	now := time.Now()
	c.mu.Lock()
	if e, ok := c.entries[key]; ok {
		select {
		case <-e.done:
			if now.Before(e.expires) {
				c.mu.Unlock()
				return e.body, nil
			}
		default: // in flight
			c.mu.Unlock()
			select {
			case <-e.done:
				return e.body, e.err
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	e := &historyCacheEntry{done: make(chan struct{})}
	cached := c.reserve(key, e, now)
	c.mu.Unlock()

	buildCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), historyBuildTimeout)
	body, err := build(buildCtx)
	cancel()

	c.mu.Lock()
	e.body, e.err = body, err
	e.expires = time.Now().Add(historyCacheTTL)
	if err != nil && cached && c.entries[key] == e {
		delete(c.entries, key)
	}
	close(e.done)
	c.mu.Unlock()
	return body, err
}

// reserve stores e under key unless the cache is full of live entries.
// Callers hold c.mu.
func (c *historyCache) reserve(key string, e *historyCacheEntry, now time.Time) bool {
	if _, ok := c.entries[key]; !ok && len(c.entries) >= historyCacheMaxEntries {
		for k, old := range c.entries {
			select {
			case <-old.done:
				if !now.Before(old.expires) {
					delete(c.entries, k)
				}
			default:
			}
		}
		if len(c.entries) >= historyCacheMaxEntries {
			return false
		}
	}
	c.entries[key] = e
	return true
}
