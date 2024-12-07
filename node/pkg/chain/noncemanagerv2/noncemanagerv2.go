package noncemanagerv2

import (
	"context"
	"errors"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/utils"
)

type NonceManagerV2 struct {
	noncePool chan uint64
	client    utils.ClientInterface
	wallet    string
	mu        sync.Mutex
}

const (
	poolSize               = 100
	minimumNoncePoolSize   = 10
	poolAutoRefillInterval = time.Minute
)

func New(ctx context.Context, client utils.ClientInterface, wallet string) (*NonceManagerV2, error) {
	if client == nil {
		return nil, errors.New("empty client")
	}

	if wallet == "" {
		return nil, errors.New("empty wallet")
	}

	pool := make(chan uint64, poolSize)
	currentNonce, err := utils.GetNonceFromPk(ctx, wallet, client)
	if err != nil {
		return nil, err
	}

	for i := uint64(0); i < poolSize; i++ {
		pool <- currentNonce + i
	}

	return &NonceManagerV2{
		noncePool: pool,
		client:    client,
		wallet:    wallet,
	}, nil
}

func (m *NonceManagerV2) GetNonce() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.noncePool) < minimumNoncePoolSize {
		m.fillPool()
	}

	nonce := <-m.noncePool
	return nonce
}

func (m *NonceManagerV2) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentNonce, err := utils.GetNonceFromPk(ctx, m.wallet, m.client)
	if err != nil {
		return err
	}

	m.flushPool()

	for i := uint64(0); i < poolSize; i++ {
		m.noncePool <- currentNonce + i
	}
	return nil
}

func (m *NonceManagerV2) fillPool() {
	nonce := <-m.noncePool
	pool := make(chan uint64, poolSize)
	for i := 0; i < poolSize; i++ {
		pool <- nonce + uint64(i)
	}
	m.noncePool = pool
}

func (m *NonceManagerV2) flushPool() {
	for len(m.noncePool) > 0 {
		<-m.noncePool
	}
}
