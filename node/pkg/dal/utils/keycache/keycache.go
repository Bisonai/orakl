package keycache

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	description string
	expiry      time.Time
}

// KeyCache caches validated API keys with their description for ttl.
type KeyCache struct {
	mu   sync.RWMutex
	keys map[string]entry
	ttl  time.Duration
}

func NewAPIKeyCache(ttl time.Duration) *KeyCache {
	return &KeyCache{
		keys: make(map[string]entry),
		ttl:  ttl,
	}
}

func (c *KeyCache) Set(key, description string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[key] = entry{description: description, expiry: time.Now().Add(c.ttl)}
}

// Get returns the cached description of key and whether it is cached and unexpired.
func (c *KeyCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, exists := c.keys[key]
	if !exists || time.Now().After(e.expiry) {
		return "", false
	}
	return e.description, true
}

func (c *KeyCache) CleanupLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.Cleanup()
			}
		}
	}()
}

func (c *KeyCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for key, e := range c.keys {
		if now.After(e.expiry) {
			delete(c.keys, key)
		}
	}
}
