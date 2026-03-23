package fetcher

import (
	"sync"
	"time"
)

const (
	CircuitBreakerThreshold = 5
	CircuitBreakerCooldown  = 5 * time.Minute
)

type feedCircuitBreaker struct {
	consecutiveFails int
	openUntil        time.Time
}

type circuitBreakerMap struct {
	mu       sync.Mutex
	breakers map[int32]*feedCircuitBreaker
}

func newCircuitBreakerMap() *circuitBreakerMap {
	return &circuitBreakerMap{
		breakers: make(map[int32]*feedCircuitBreaker),
	}
}

func (m *circuitBreakerMap) isOpen(feedID int32) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cb, exists := m.breakers[feedID]
	if !exists {
		return false
	}
	if cb.consecutiveFails < CircuitBreakerThreshold {
		return false
	}
	return time.Now().Before(cb.openUntil)
}

func (m *circuitBreakerMap) recordSuccess(feedID int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cb, exists := m.breakers[feedID]
	if !exists {
		return
	}
	cb.consecutiveFails = 0
}

func (m *circuitBreakerMap) recordFailure(feedID int32, feedName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cb, exists := m.breakers[feedID]
	if !exists {
		cb = &feedCircuitBreaker{}
		m.breakers[feedID] = cb
	}
	cb.consecutiveFails++
	if cb.consecutiveFails == CircuitBreakerThreshold {
		cb.openUntil = time.Now().Add(CircuitBreakerCooldown)
		return true // circuit just opened
	}
	return false
}
