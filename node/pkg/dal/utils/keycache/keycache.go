package keycache

import (
	"sync"
	"time"
)

type KeyCache struct {
	mu   sync.RWMutex
	keys map[string]time.Time
	ttl  time.Duration
}

func NewAPIKeyCache(ttl time.Duration) *KeyCache {
	return &KeyCache{
		keys: make(map[string]time.Time),
		ttl:  ttl,
	}
}

func (c *KeyCache) Set(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys[key] = time.Now().Add(c.ttl)
}

func (c *KeyCache) Get(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	expiry, exists := c.keys[key]
	if !exists || time.Now().After(expiry) {
		return false
	}
	return true
}

func (c *KeyCache) CleanupLoop(interval time.Duration) {
	go func() {
		time.Sleep(interval)
		c.Cleanup()
	}()
}

func (c *KeyCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for key, expiry := range c.keys {
		if now.After(expiry) {
			delete(c.keys, key)
		}
	}
}
