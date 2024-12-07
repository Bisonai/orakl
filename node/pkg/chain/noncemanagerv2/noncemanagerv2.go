package noncemanagerv2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
	"github.com/rs/zerolog/log"
)

type NonceManagerV2 struct {
	mu        sync.Mutex
	noncePool map[string]chan uint64 // address -> nonce pool channel
	client    utils.ClientInterface
}

const (
	poolSize               = 15 // expect maximum 15 submission per minute
	minimumNoncePoolSize   = 3
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
		m.noncePool[address] = make(chan uint64, 30)
		if err := m.unsafeRefill(ctx, address); err != nil {
			return 0, err
		}
	}
	if len(m.noncePool[address]) < minimumNoncePoolSize {
		if err := m.unsafeRefill(ctx, address); err != nil {
			return 0, fmt.Errorf("failed to refill nonce pool: %w", err)
		}
	}

	nonce := <-m.noncePool[address]
	return nonce, nil
}

func (m *NonceManagerV2) unsafeRefill(ctx context.Context, address string) error {
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

func (m *NonceManagerV2) refillAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for address := range m.noncePool {
		if err := m.unsafeRefill(ctx, address); err != nil {
			return err
		}
	}
	return nil
}

func (m *NonceManagerV2) StartAutoRefill(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(poolAutoRefillInterval):
			err := m.refillAll(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Failed to refill nonce pool")
			}
		}
	}
}
