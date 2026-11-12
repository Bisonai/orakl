package websocketchainreader

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/eth_client"
	"bisonai.com/miko/node/pkg/chain/utils"
	kaiaSdk "github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/client"
	"github.com/stretchr/testify/assert"
)

type stubClient struct {
	utils.ClientInterface
	closed bool
}

func (s *stubClient) Close() { s.closed = true }

func TestCloseSkipsUnsetClients(t *testing.T) {
	eth := &stubClient{}
	kaia := &stubClient{}
	c := &ChainReader{
		EthClient:      eth,
		KaiaClient:     kaia,
		BaseClient:     (*eth_client.EthClient)(nil), // optional chain, unset
		ArbitrumClient: (*eth_client.EthClient)(nil),
		BscClient:      (*client.Client)(nil),
	}
	assert.NotPanics(t, c.Close)
	assert.True(t, eth.closed)
	assert.True(t, kaia.closed)
}

type fakeSub struct {
	errCh        chan error
	unsubscribed chan struct{}
	once         sync.Once
}

func newFakeSub() *fakeSub {
	return &fakeSub{errCh: make(chan error, 1), unsubscribed: make(chan struct{})}
}

func (s *fakeSub) Err() <-chan error { return s.errCh }
func (s *fakeSub) Unsubscribe()      { s.once.Do(func() { close(s.unsubscribed) }) }

func (s *fakeSub) isUnsubscribed() bool {
	select {
	case <-s.unsubscribed:
		return true
	default:
		return false
	}
}

type subscribeCall struct {
	query kaiaSdk.FilterQuery
	logs  chan<- types.Log
	sub   *fakeSub
}

// fakeLogClient hands each subscription to the test, which pushes logs and
// errors into it.
type fakeLogClient struct {
	utils.ClientInterface
	head  atomic.Uint64
	calls chan subscribeCall
}

func newFakeClient(head uint64) *fakeLogClient {
	f := &fakeLogClient{calls: make(chan subscribeCall, 4)}
	f.head.Store(head)
	return f
}

func (f *fakeLogClient) BlockNumber(ctx context.Context) (*big.Int, error) {
	return new(big.Int).SetUint64(f.head.Load()), nil
}

func (f *fakeLogClient) SubscribeFilterLogs(ctx context.Context, q kaiaSdk.FilterQuery, ch chan<- types.Log) (kaiaSdk.Subscription, error) {
	sub := newFakeSub()
	f.calls <- subscribeCall{query: q, logs: ch, sub: sub}
	return sub, nil
}

func nextCall(t *testing.T, calls <-chan subscribeCall) subscribeCall {
	t.Helper()
	select {
	case c := <-calls:
		return c
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SubscribeFilterLogs")
		return subscribeCall{}
	}
}

func recvLog(t *testing.T, out <-chan types.Log) types.Log {
	t.Helper()
	select {
	case l := <-out:
		return l
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for log")
		return types.Log{}
	}
}

func subscribe(t *testing.T, ctx context.Context, fc *fakeLogClient, out chan types.Log, opts ...SubscribeOption) {
	t.Helper()
	r := &ChainReader{EthClient: fc, RetryPeriod: time.Millisecond}
	opts = append([]SubscribeOption{
		WithAddress("0x0000000000000000000000000000000000000001"),
		WithChannel(out),
		WithChainType(Ethereum),
	}, opts...)
	assert.NoError(t, r.Subscribe(ctx, opts...))
}

// TestResubscribeStartsFromHead: the configured start block applies to the
// first subscribe only (#2791); the old subscription is released first.
func TestResubscribeStartsFromHead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(20)
	out := make(chan types.Log)
	subscribe(t, ctx, fc, out, WithStartBlockNumber(big.NewInt(10)))

	first := nextCall(t, fc.calls)
	assert.Equal(t, uint64(10), first.query.FromBlock.Uint64())

	fc.head.Store(30)
	first.sub.errCh <- errors.New("connection dropped")

	second := nextCall(t, fc.calls)
	assert.True(t, first.sub.isUnsubscribed(), "old subscription not released before resubscribe")
	assert.Equal(t, uint64(30), second.query.FromBlock.Uint64())

	// the new subscription delivers
	second.logs <- types.Log{BlockNumber: 31}
	assert.Equal(t, uint64(31), recvLog(t, out).BlockNumber)
}

func TestRemovedLogNotForwarded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(1)
	out := make(chan types.Log)
	subscribe(t, ctx, fc, out)

	call := nextCall(t, fc.calls)
	go func() {
		for _, l := range []types.Log{{BlockNumber: 2, Index: 0, Removed: true}, {BlockNumber: 2, Index: 1}} {
			select {
			case call.logs <- l:
			case <-ctx.Done():
				return
			}
		}
	}()
	got := recvLog(t, out)
	assert.False(t, got.Removed, "removed log forwarded")
	assert.Equal(t, uint(1), got.Index)
}

func TestProcessLogsExitsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		processLogs(ctx, newFakeSub(), make(chan types.Log), make(chan types.Log))
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("processLogs did not exit after ctx cancel")
	}
}

func TestHandleSubscriptionUnsubscribesOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fc := newFakeClient(1)
	subscribe(t, ctx, fc, make(chan types.Log))

	call := nextCall(t, fc.calls)
	cancel()
	select {
	case <-call.sub.unsubscribed:
	case <-time.After(time.Second):
		t.Fatal("subscription not unsubscribed after ctx cancel")
	}
}
