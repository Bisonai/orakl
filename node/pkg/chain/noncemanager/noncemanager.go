package noncemanager

import (
	"fmt"
	"sync"
)

type NonceManager struct {
	mu     sync.RWMutex
	nonces map[string]uint64
}

func New() *NonceManager {
	return &NonceManager{
		nonces: make(map[string]uint64),
	}
}

func (m *NonceManager) SetNonce(address string, nonce uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nonces[address] = nonce
}

func (m *NonceManager) GetNonceAndIncrement(address string) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nonce, ok := m.nonces[address]
	if !ok {
		return 0, fmt.Errorf("nonce not found")
	}

	nonce++
	m.nonces[address] = nonce
	return nonce, nil
}

func (m *NonceManager) GetNonce(address string) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result, ok := m.nonces[address]
	if !ok {
		return 0, fmt.Errorf("nonce not found")
	}
	return result, nil
}
