package fetcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker_OpensAtThreshold(t *testing.T) {
	m := newCircuitBreakerMap()
	const feedID int32 = 1

	for i := 1; i < CircuitBreakerThreshold; i++ {
		opened := m.recordFailure(feedID, "test")
		assert.False(t, opened, "should not open before threshold (fail %d)", i)
		assert.False(t, m.isOpen(feedID))
	}

	opened := m.recordFailure(feedID, "test")
	assert.True(t, opened, "should open at threshold")
	assert.True(t, m.isOpen(feedID))
}

func TestCircuitBreaker_RecordSuccessResets(t *testing.T) {
	m := newCircuitBreakerMap()
	const feedID int32 = 1

	for i := 0; i < CircuitBreakerThreshold-1; i++ {
		m.recordFailure(feedID, "test")
	}
	m.recordSuccess(feedID)

	// Counter reset, so next single failure should not open.
	opened := m.recordFailure(feedID, "test")
	assert.False(t, opened, "counter should have reset on success")
}

// Regression test for the bug where the circuit breaker permanently stayed
// closed after the first cooldown expired, because recordFailure used `==`
// instead of `>=` and never re-armed on subsequent failures.
func TestCircuitBreaker_ReArmsAfterCooldown(t *testing.T) {
	m := newCircuitBreakerMap()
	const feedID int32 = 1

	// Drive to the threshold to open the breaker.
	for i := 0; i < CircuitBreakerThreshold; i++ {
		m.recordFailure(feedID, "test")
	}
	assert.True(t, m.isOpen(feedID))

	// Force the cooldown to be in the past, simulating cooldown expiry.
	m.breakers[feedID].openUntil = time.Now().Add(-time.Second)
	assert.False(t, m.isOpen(feedID), "breaker should close after cooldown expires")

	// A fresh failure must re-arm the breaker, not silently let traffic through.
	opened := m.recordFailure(feedID, "test")
	assert.True(t, opened, "breaker must re-arm on the next failure after cooldown")
	assert.True(t, m.isOpen(feedID))
}

func TestCircuitBreaker_StaysOpenDuringCooldown(t *testing.T) {
	m := newCircuitBreakerMap()
	const feedID int32 = 1

	for i := 0; i < CircuitBreakerThreshold; i++ {
		m.recordFailure(feedID, "test")
	}

	// Additional failures during cooldown must not return "opened" again.
	for i := 0; i < 10; i++ {
		opened := m.recordFailure(feedID, "test")
		assert.False(t, opened, "should not log opened again while still in cooldown")
	}
	assert.True(t, m.isOpen(feedID))
}
