package noncemanager

import (
	"context"
	"fmt"
	"sync"

	"bisonai.com/miko/node/pkg/chain/utils"
)

type NonceManager struct {
	mu     sync.RWMutex
	nonces map[string]uint64
}

var (
	Manager *NonceManager
	once    sync.Once
)

func Get() *NonceManager {
	once.Do(func() {
		Manager = &NonceManager{
			nonces: make(map[string]uint64),
			mu:     sync.RWMutex{},
		}
	})
	return Manager
}

func ResetNonce(ctx context.Context, address string, client utils.ClientInterface) error {
	return Get().ResetNonce(ctx, address, client)
}

func Set(address string, nonce uint64) {
	Get().SetNonce(address, nonce)
}

func GetAndIncrementNonce(address string) (uint64, error) {
	return Get().GetAndIncrementNonce(address)
}

func (m *NonceManager) ResetNonce(ctx context.Context, address string, client utils.ClientInterface) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	nonce, err := utils.GetNonceFromPk(ctx, address, client)
	if err != nil {
		return err
	}
	m.nonces[address] = nonce
	return nil
}

func (m *NonceManager) SetNonce(address string, nonce uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nonces[address] = nonce
}

func (m *NonceManager) GetAndIncrementNonce(address string) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	nonce, ok := m.nonces[address]
	if !ok {
		return 0, fmt.Errorf("nonce not found")
	}

	m.nonces[address] = nonce + 1
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

// ResetInstance is used for testing purposes to reset the singleton instance
func ResetInstance() {
	Manager = nil
	once = sync.Once{}
}
