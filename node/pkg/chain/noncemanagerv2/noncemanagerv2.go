package noncemanagerv2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
)

type NonceManagerV2 struct {
	mu        sync.Mutex
	noncePool map[string]chan uint64 // address -> nonce pool channel
	client    utils.ClientInterface
}

const (
	poolSize               = 100 // expect maximum 15 submission per minute
	minimumNoncePoolSize   = 5
	poolAutoRefillInterval = time.Minute
)

func New(client utils.ClientInterface) *NonceManagerV2 {
	return &NonceManagerV2{
		noncePool: make(map[string]chan uint64),
		client:    client,
	}
}

func (m *NonceManagerV2) GetNonce(ctx context.Context, address string) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.noncePool[address]; !ok {
		m.noncePool[address] = make(chan uint64, poolSize)
		if err := m.unsafeInitPool(ctx, address); err != nil {
			return 0, fmt.Errorf("failed to refill nonce pool: %w", err)
		}
	}

	if len(m.noncePool[address]) < minimumNoncePoolSize {
		m.unsafeFillPool(address)
	}

	nonce := <-m.noncePool[address]
	return nonce, nil
}

func (m *NonceManagerV2) Reset(ctx context.Context, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.unsafeInitPool(ctx, address)
}

func (m *NonceManagerV2) unsafeFillPool(address string) {
	nonce := <-m.noncePool[address]
	for i := 0; i < poolSize-len(m.noncePool[address]); i++ {
		m.noncePool[address] <- nonce + uint64(i)
	}
}

func (m *NonceManagerV2) unsafeInitPool(ctx context.Context, address string) error {
	currentNonce, err := utils.GetNonceFromPk(ctx, address, m.client)
	if err != nil {
		return err
	}

	m.unsafeFlushPool(address)
	for i := uint64(0); i < poolSize; i++ {
		m.noncePool[address] <- currentNonce + i
	}
	return nil
}

func (m *NonceManagerV2) unsafeFlushPool(address string) {
	if pool, exists := m.noncePool[address]; exists {
		for len(pool) > 0 {
			<-pool
		}
	}
}
