package noncemanagerv2

import (
	"context"
	"encoding/hex"
	"math/big"
	"sync"
	"testing"

	"github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClient only serves PendingNonceAt.
type fakeClient struct {
	mu    sync.Mutex
	nonce uint64
}

func (c *fakeClient) setNonce(n uint64) { c.mu.Lock(); c.nonce = n; c.mu.Unlock() }

func (c *fakeClient) Close() {}
func (c *fakeClient) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nonce, nil
}
func (c *fakeClient) SuggestGasPrice(context.Context) (*big.Int, error)          { return nil, nil }
func (c *fakeClient) EstimateGas(context.Context, kaia.CallMsg) (uint64, error)  { return 0, nil }
func (c *fakeClient) SendTransaction(context.Context, *types.Transaction) error { return nil }
func (c *fakeClient) CallContract(context.Context, kaia.CallMsg, *big.Int) ([]byte, error) {
	return nil, nil
}
func (c *fakeClient) NetworkID(context.Context) (*big.Int, error) { return nil, nil }
func (c *fakeClient) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return nil, nil
}
func (c *fakeClient) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	return nil, nil
}
func (c *fakeClient) BlockNumber(context.Context) (*big.Int, error) { return nil, nil }
func (c *fakeClient) SubscribeFilterLogs(context.Context, kaia.FilterQuery, chan<- types.Log) (kaia.Subscription, error) {
	return nil, nil
}
func (c *fakeClient) SubscribeNewHead(context.Context, chan<- *types.Header) (kaia.Subscription, error) {
	return nil, nil
}

func newTestManager(t *testing.T, start uint64) (*NonceManagerV2, *fakeClient) {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	client := &fakeClient{nonce: start}
	m, err := New(context.Background(), client, hex.EncodeToString(crypto.FromECDSA(key)))
	require.NoError(t, err)
	return m, client
}

func TestReleaseReusesNonce(t *testing.T) {
	m, _ := newTestManager(t, 50)

	leases := make([]Lease, 5)
	for i := range leases {
		leases[i] = m.Acquire()
		assert.Equal(t, uint64(50+i), leases[i].Nonce)
	}

	// build failed for 52 and 51: lowest released first, then the pool resumes
	m.Release(leases[2])
	m.Release(leases[1])
	assert.Equal(t, uint64(51), m.GetNonce())
	assert.Equal(t, uint64(52), m.GetNonce())
	assert.Equal(t, uint64(55), m.GetNonce())
}

func TestReleaseIgnoresDuplicateAndUnissued(t *testing.T) {
	m, _ := newTestManager(t, 0)
	l := m.Acquire()
	m.Release(l)
	m.Release(l)
	m.Release(Lease{Nonce: 7}) // not handed out yet
	assert.Equal(t, uint64(0), m.GetNonce())
	assert.Equal(t, uint64(1), m.GetNonce())
	assert.Equal(t, uint64(2), m.GetNonce())
}

func TestReleaseAfterResetIgnored(t *testing.T) {
	m, client := newTestManager(t, 10)
	old := m.Acquire() // 10
	_ = m.Acquire()    // 11

	client.setNonce(10)
	require.NoError(t, m.Reset(context.Background()))

	// 10 is back in the re-seeded pool; releasing the stale lease must not
	// hand it out twice
	m.Release(old)
	assert.Equal(t, uint64(10), m.GetNonce())
	assert.Equal(t, uint64(11), m.GetNonce())
}

func TestReleaseAcrossFillPool(t *testing.T) {
	m, _ := newTestManager(t, 0)
	var held []Lease
	for i := 0; i < poolSize; i++ {
		held = append(held, m.Acquire())
	}
	m.Release(held[3])
	assert.Equal(t, uint64(3), m.GetNonce())
	assert.Equal(t, uint64(poolSize), m.GetNonce())
}

func TestConcurrentAcquireRelease(t *testing.T) {
	m, _ := newTestManager(t, 0)

	const workers = 16
	const perWorker = 200
	var mu sync.Mutex
	kept := map[uint64]int{}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				l := m.Acquire()
				if (i+w)%3 == 0 {
					m.Release(l) // simulated build failure
					continue
				}
				mu.Lock()
				kept[l.Nonce]++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	// every kept nonce is unique and they form 0..len-1 with no gap once the
	// remaining released nonces are taken
	for len(m.released) > 0 {
		kept[m.GetNonce()]++
	}
	for n, c := range kept {
		assert.Equal(t, 1, c, "nonce %d handed out %d times", n, c)
	}
	for n := uint64(0); n < uint64(len(kept)); n++ {
		_, ok := kept[n]
		assert.True(t, ok, "gap at nonce %d", n)
	}
}
