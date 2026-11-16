package noncemanagerv2

import (
	"context"
	"errors"
	"sort"
	"sync"

	"bisonai.com/miko/node/pkg/chain/utils"
)

type NonceManagerV2 struct {
	noncePool chan uint64
	client    utils.ClientInterface
	wallet    string
	mu        sync.Mutex

	// released holds nonces handed back via Release (tx never broadcast),
	// sorted ascending. They are handed out again before the pool.
	released []uint64
	// next is the next nonce the pool would hand out; every released
	// nonce of the current generation is below it.
	next uint64
	// gen changes on Reset so a lease taken before a reset can't be
	// released into the re-seeded pool (it could duplicate a pool nonce).
	gen uint64
}

// Lease is a nonce handed out by Acquire. Pass it to Release only when the
// tx using it was certainly not broadcast.
type Lease struct {
	Nonce uint64
	gen   uint64
}

const (
	poolSize             = 100
	minimumNoncePoolSize = 10
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
		next:      currentNonce,
	}, nil
}

func (m *NonceManagerV2) GetNonce() uint64 {
	return m.Acquire().Nonce
}

// Acquire returns the lowest released nonce if any, else the next pool nonce.
func (m *NonceManagerV2) Acquire() Lease {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.released) > 0 {
		nonce := m.released[0]
		m.released = m.released[1:]
		return Lease{Nonce: nonce, gen: m.gen}
	}

	if len(m.noncePool) < minimumNoncePoolSize {
		m.fillPool()
	}

	nonce := <-m.noncePool
	m.next = nonce + 1
	return Lease{Nonce: nonce, gen: m.gen}
}

// Release hands back a nonce whose tx was never broadcast, so the next
// Acquire reuses it instead of leaving a gap. Leases from before a Reset,
// nonces not yet handed out and duplicates are ignored.
func (m *NonceManagerV2) Release(l Lease) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if l.gen != m.gen || l.Nonce >= m.next {
		return
	}

	i := sort.Search(len(m.released), func(i int) bool { return m.released[i] >= l.Nonce })
	if i < len(m.released) && m.released[i] == l.Nonce {
		return
	}
	m.released = append(m.released, 0)
	copy(m.released[i+1:], m.released[i:])
	m.released[i] = l.Nonce
}

func (m *NonceManagerV2) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentNonce, err := utils.GetNonceFromPk(ctx, m.wallet, m.client)
	if err != nil {
		return err
	}

	m.flushPool()
	m.released = nil
	m.next = currentNonce
	m.gen++

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
