package keycache

import (
	"context"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
)

type KeyCache struct {
	mu   sync.RWMutex
	keys map[string]time.Time
	ttl  time.Duration
}

type DBKeyResult struct {
	Exist bool `db:"exists"`
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

func ValidateApiKeyFromDB(ctx context.Context, apiKey string) bool {
	res, err := db.QueryRow[DBKeyResult](ctx, "SELECT true as exists FROM keys WHERE key = @key", map[string]any{"key": apiKey})
	if err != nil {
		log.Error().Err(err).Msg("Error validating API key")
	}
	return res.Exist && err == nil
}
