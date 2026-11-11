package websocketchainreader

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/eth_client"
	"bisonai.com/miko/node/pkg/chain/utils"
	kaiaSdk "github.com/kaiachain/kaia"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/client"
	kaiaCommon "github.com/kaiachain/kaia/common"
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

type subscribeCall struct {
	query kaiaSdk.FilterQuery
	logs  chan<- types.Log
	sub   *fakeSub
}

type fakeLogClient struct {
	utils.ClientInterface
	head  *big.Int
	calls chan subscribeCall
}

func (f *fakeLogClient) BlockNumber(ctx context.Context) (*big.Int, error) {
	return new(big.Int).Set(f.head), nil
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

func newTestReader(fc *fakeLogClient) *ChainReader {
	return &ChainReader{EthClient: fc, RetryPeriod: time.Millisecond}
}

func testLog(block uint64, index uint) types.Log {
	return types.Log{BlockNumber: block, Index: index, TxHash: kaiaCommon.BigToHash(new(big.Int).SetUint64(block*100 + uint64(index)))}
}

func TestResubscribeResumesFromLastDeliveredBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := &fakeLogClient{head: big.NewInt(1000), calls: make(chan subscribeCall, 4)}
	out := make(chan types.Log)
	err := newTestReader(fc).Subscribe(ctx,
		WithAddress("0x0000000000000000000000000000000000000001"),
		WithChannel(out),
		WithChainType(Ethereum),
		WithStartBlockNumber(big.NewInt(10)))
	assert.NoError(t, err)

	first := nextCall(t, fc.calls)
	assert.Equal(t, int64(10), first.query.FromBlock.Int64())

	first.logs <- testLog(12, 0)
	recvLog(t, out)
	first.logs <- testLog(15, 1)
	recvLog(t, out)

	fc.head = big.NewInt(2000) // head moved during outage; must not be used
	first.sub.errCh <- errors.New("connection dropped")

	second := nextCall(t, fc.calls)
	select {
	case <-first.sub.unsubscribed:
	default:
		t.Fatal("old subscription not unsubscribed before resubscribe")
	}
	// Inclusive resume: last delivered block, not start block and not head.
	assert.Equal(t, int64(15), second.query.FromBlock.Int64())

	// Replayed log of block 15 is dropped; a new log of the same block and later blocks pass.
	second.logs <- testLog(15, 1)
	second.logs <- testLog(15, 2)
	assert.Equal(t, testLog(15, 2), recvLog(t, out))
	second.logs <- testLog(16, 0)
	assert.Equal(t, testLog(16, 0), recvLog(t, out))
}

func TestResubscribeWithoutLogsUsesInitialHead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := &fakeLogClient{head: big.NewInt(500), calls: make(chan subscribeCall, 4)}
	out := make(chan types.Log)
	assert.NoError(t, newTestReader(fc).Subscribe(ctx,
		WithAddress("0x0000000000000000000000000000000000000001"),
		WithChannel(out)))

	first := nextCall(t, fc.calls)
	assert.Equal(t, int64(500), first.query.FromBlock.Int64())

	fc.head = big.NewInt(600)
	first.sub.errCh <- errors.New("connection dropped")

	second := nextCall(t, fc.calls)
	assert.Equal(t, int64(500), second.query.FromBlock.Int64())
}

func TestProcessLogsExitsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sub := newFakeSub()
	done := make(chan struct{})
	go func() {
		processLogs(ctx, sub, make(chan types.Log), make(chan types.Log), &logCursor{})
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
	fc := &fakeLogClient{head: big.NewInt(1), calls: make(chan subscribeCall, 1)}
	assert.NoError(t, newTestReader(fc).Subscribe(ctx,
		WithAddress("0x0000000000000000000000000000000000000001"),
		WithChannel(make(chan types.Log))))

	call := nextCall(t, fc.calls)
	cancel()
	select {
	case <-call.sub.unsubscribed:
	case <-time.After(time.Second):
		t.Fatal("subscription not unsubscribed after ctx cancel")
	}
}
