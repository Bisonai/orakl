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

// Backfill relies on the real clients exposing FilterLogs.
var (
	_ logFilterer = (*eth_client.EthClient)(nil)
	_ logFilterer = (*client.Client)(nil)
)

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

// fakeLogClient models a node: the subscription never replays history (it only
// gets what the test pushes as newly mined), FilterLogs serves the chain.
type fakeLogClient struct {
	utils.ClientInterface
	mu        sync.Mutex
	head      uint64
	chain     []types.Log
	filterErr error
	filterQs  []kaiaSdk.FilterQuery
	calls     chan subscribeCall
}

func (f *fakeLogClient) BlockNumber(ctx context.Context) (*big.Int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return new(big.Int).SetUint64(f.head), nil
}

func (f *fakeLogClient) SubscribeFilterLogs(ctx context.Context, q kaiaSdk.FilterQuery, ch chan<- types.Log) (kaiaSdk.Subscription, error) {
	sub := newFakeSub()
	f.calls <- subscribeCall{query: q, logs: ch, sub: sub}
	return sub, nil
}

func (f *fakeLogClient) FilterLogs(ctx context.Context, q kaiaSdk.FilterQuery) ([]types.Log, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filterQs = append(f.filterQs, q)
	if f.filterErr != nil {
		return nil, f.filterErr
	}
	var out []types.Log
	for _, l := range f.chain {
		if l.BlockNumber >= q.FromBlock.Uint64() && l.BlockNumber <= q.ToBlock.Uint64() {
			out = append(out, l)
		}
	}
	return out, nil
}

// mine appends logs to the chain and moves the head.
func (f *fakeLogClient) mine(head uint64, logs ...types.Log) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chain = append(f.chain, logs...)
	f.head = head
}

func (f *fakeLogClient) filterQueries() []kaiaSdk.FilterQuery {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]kaiaSdk.FilterQuery(nil), f.filterQs...)
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

func assertNoLog(t *testing.T, out <-chan types.Log) {
	t.Helper()
	select {
	case l := <-out:
		t.Fatalf("unexpected log %d/%d", l.BlockNumber, l.Index)
	case <-time.After(50 * time.Millisecond):
	}
}

func newFakeClient(head uint64) *fakeLogClient {
	return &fakeLogClient{head: head, calls: make(chan subscribeCall, 4)}
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

func testLog(block uint64, index uint) types.Log {
	return types.Log{BlockNumber: block, Index: index, TxHash: kaiaCommon.BigToHash(new(big.Int).SetUint64(block*100 + uint64(index)))}
}

func TestResubscribeResumesFromLastDeliveredBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(20)
	fc.mine(20, testLog(12, 0), testLog(15, 1))
	out := make(chan types.Log)
	subscribe(t, ctx, fc, out, WithStartBlockNumber(big.NewInt(10)))

	first := nextCall(t, fc.calls)
	assert.Equal(t, uint64(10), first.query.FromBlock.Uint64())
	assert.Equal(t, testLog(12, 0), recvLog(t, out)) // backfilled
	assert.Equal(t, testLog(15, 1), recvLog(t, out))

	fc.mine(21, testLog(21, 0))
	first.logs <- testLog(21, 0) // streamed
	assert.Equal(t, testLog(21, 0), recvLog(t, out))

	// Outage: later log of block 21 and block 25 mined while disconnected.
	fc.mine(30, testLog(21, 1), testLog(25, 0))
	first.sub.errCh <- errors.New("connection dropped")

	second := nextCall(t, fc.calls)
	select {
	case <-first.sub.unsubscribed:
	default:
		t.Fatal("old subscription not unsubscribed before resubscribe")
	}
	// Inclusive resume: last delivered block, not start block (10) and not head (30).
	assert.Equal(t, uint64(21), second.query.FromBlock.Uint64())

	// Outage logs come from the backfill; already-delivered 21/0 is not resent.
	assert.Equal(t, testLog(21, 1), recvLog(t, out))
	assert.Equal(t, testLog(25, 0), recvLog(t, out))

	// Seam duplicate from the stream is dropped; new logs pass.
	second.logs <- testLog(25, 0)
	second.logs <- testLog(31, 0)
	assert.Equal(t, testLog(31, 0), recvLog(t, out))

	qs := fc.filterQueries()
	last := qs[len(qs)-1]
	assert.Equal(t, uint64(21), last.FromBlock.Uint64())
	assert.Equal(t, uint64(30), last.ToBlock.Uint64())
}

func TestResubscribeWithoutLogsUsesInitialHead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(500)
	fc.mine(500, testLog(500, 0)) // mined before subscribe: not delivered
	out := make(chan types.Log)
	subscribe(t, ctx, fc, out)

	first := nextCall(t, fc.calls)
	assert.Equal(t, uint64(501), first.query.FromBlock.Uint64())

	fc.mine(600, testLog(550, 0))
	first.sub.errCh <- errors.New("connection dropped")

	second := nextCall(t, fc.calls)
	assert.Equal(t, uint64(501), second.query.FromBlock.Uint64())
	assert.Equal(t, testLog(550, 0), recvLog(t, out))
	assertNoLog(t, out)
}

func TestBackfillCappedAndChunked(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(0)
	out := make(chan types.Log)
	head := uint64(maxBackfillBlocks + 5000)
	fc.mine(head, testLog(10, 0), testLog(head, 0))
	subscribe(t, ctx, fc, out, WithStartBlockNumber(big.NewInt(1)))

	nextCall(t, fc.calls)
	assert.Equal(t, testLog(head, 0), recvLog(t, out)) // block 10 is beyond the cap

	qs := fc.filterQueries()
	assert.Len(t, qs, maxBackfillBlocks/backfillChunkBlocks)
	assert.Equal(t, head-maxBackfillBlocks+1, qs[0].FromBlock.Uint64())
	for _, q := range qs {
		assert.LessOrEqual(t, q.ToBlock.Uint64()-q.FromBlock.Uint64()+1, uint64(backfillChunkBlocks))
	}
	assert.Equal(t, head, qs[len(qs)-1].ToBlock.Uint64())
}

func TestBackfillErrorRetriesThenStreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(5)
	fc.filterErr = errors.New("rate limited")
	out := make(chan types.Log)
	subscribe(t, ctx, fc, out, WithStartBlockNumber(big.NewInt(1)))

	var call subscribeCall
	for i := 0; i < maxBackfillAttempts; i++ {
		call = nextCall(t, fc.calls)
		assert.Equal(t, uint64(1), call.query.FromBlock.Uint64())
	}
	call.logs <- testLog(6, 0)
	assert.Equal(t, testLog(6, 0), recvLog(t, out))
}

func TestBackfillLaggingBackendDoesNotDropStreamedLogs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fc := newFakeClient(100)
	out := make(chan types.Log)
	subscribe(t, ctx, fc, out)

	first := nextCall(t, fc.calls)
	// Head moves to 110 but the getLogs backend hasn't indexed block 110 yet.
	fc.mine(110)
	first.sub.errCh <- errors.New("connection dropped")
	second := nextCall(t, fc.calls)
	second.logs <- testLog(110, 0)
	assert.Equal(t, testLog(110, 0), recvLog(t, out))
}

func withHash(l types.Log, h byte, removed bool) types.Log {
	l.BlockHash = kaiaCommon.Hash{h}
	l.Removed = removed
	return l
}

func TestReorgedLogDeliveredInEitherOrder(t *testing.T) {
	for _, removedFirst := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		fc := newFakeClient(100)
		out := make(chan types.Log)
		subscribe(t, ctx, fc, out)

		call := nextCall(t, fc.calls)
		orig := withHash(testLog(101, 0), 1, false)
		removed := withHash(testLog(101, 0), 1, true)
		remined := withHash(testLog(101, 0), 2, false)

		call.logs <- orig
		assert.Equal(t, orig, recvLog(t, out))
		if removedFirst {
			call.logs <- removed
			call.logs <- remined
			assert.Equal(t, remined, recvLog(t, out))
		} else {
			call.logs <- remined
			assert.Equal(t, remined, recvLog(t, out))
			call.logs <- removed
		}
		assertNoLog(t, out) // removed log never forwarded
		cancel()
	}
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
