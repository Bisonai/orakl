//nolint:all
package test

import (
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal/utils/keycache"
)

func TestKeyCache_SetAndGet(t *testing.T) {
	cache := keycache.NewAPIKeyCache(100 * time.Millisecond)

	// Test setting and getting a key
	key := "test-key"
	cache.Set(key)
	if !cache.Get(key) {
		t.Errorf("Expected key %s to be in cache", key)
	}

	// Wait for the key to expire
	time.Sleep(200 * time.Millisecond)
	if cache.Get(key) {
		t.Errorf("Expected key %s to have expired from cache", key)
	}
}

func TestKeyCache_Cleanup(t *testing.T) {
	cache := keycache.NewAPIKeyCache(100 * time.Millisecond)

	// Set a key and wait for it to expire
	key := "test-key"
	cache.Set(key)
	time.Sleep(200 * time.Millisecond)

	// Run cleanup and check if the key is removed
	cache.Cleanup()
	if cache.Get(key) {
		t.Errorf("Expected key %s to be removed by cleanup", key)
	}
}

func TestKeyCache_CleanupLoop(t *testing.T) {
	cache := keycache.NewAPIKeyCache(50 * time.Millisecond)

	// Set a key and wait for it to expire
	key := "test-key"
	cache.Set(key)
	cache.CleanupLoop(25 * time.Millisecond)
	time.Sleep(75 * time.Millisecond)

	// Check if the key has been removed
	if cache.Get(key) {
		t.Errorf("Expected key %s to be removed by cleanup loop", key)
	}
}

func TestKeyCache_Concurrency(t *testing.T) {
	cache := keycache.NewAPIKeyCache(1 * time.Second)

	key := "test-key"

	// Set a key in a separate goroutine
	go cache.Set(key)

	time.Sleep(10 * time.Millisecond)

	// Try getting the key in a separate goroutine
	go func() {
		if !cache.Get(key) {
			t.Errorf("Expected key %s to be in cache", key)
		}
	}()

	// Wait to ensure goroutines have completed
	time.Sleep(100 * time.Millisecond)
}
