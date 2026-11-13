package collector

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/aggregator"
	chainutils "bisonai.com/miko/node/pkg/chain/utils"
	dalcommon "bisonai.com/miko/node/pkg/dal/common"
	kaiacommon "github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func submission(symbol string, ts time.Time, proof []byte) *aggregator.SubmissionData {
	return &aggregator.SubmissionData{
		Symbol:          symbol,
		GlobalAggregate: aggregator.GlobalAggregate{Value: 100, Timestamp: ts},
		Proof:           aggregator.Proof{Proof: proof},
	}
}

// fakeOutgoing converts without verification, blocking data whose timestamp has a gate until
// the gate is closed; entered is signalled once the conversion started.
type fakeOutgoing struct {
	mu      sync.Mutex
	gates   map[int64]chan struct{}
	entered chan int64
}

func (f *fakeOutgoing) convert(_ context.Context, data *aggregator.SubmissionData) (*dalcommon.OutgoingSubmissionData, error) {
	ms := data.GlobalAggregate.Timestamp.UnixMilli()
	f.mu.Lock()
	gate := f.gates[ms]
	f.mu.Unlock()
	f.entered <- ms
	if gate != nil {
		<-gate
	}
	return &dalcommon.OutgoingSubmissionData{Symbol: data.Symbol, AggregateTime: strconv.FormatInt(ms, 10)}, nil
}

func drain(stream chan *dalcommon.OutgoingSubmissionData) []string {
	var out []string
	for {
		select {
		case d := <-stream:
			out = append(out, d.AggregateTime)
		default:
			return out
		}
	}
}

// The older message finishes verification last: it must not overwrite the newer one.
func TestOlderDataVerifiedLastDoesNotOverwrite(t *testing.T) {
	c, _ := newTestCollector(t, "A-B")
	delete(c.LatestData, "A-B")
	gate := make(chan struct{})
	t1 := time.UnixMilli(1_700_000_000_000)
	t2 := t1.Add(time.Second)
	f := &fakeOutgoing{gates: map[int64]chan struct{}{t1.UnixMilli(): gate}, entered: make(chan int64, 2)}
	c.toOutgoing = f.convert
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.processIncomingData(ctx, submission("A-B", t1, nil))
	}()
	require.Equal(t, t1.UnixMilli(), <-f.entered)

	c.processIncomingData(ctx, submission("A-B", t2, nil))
	close(gate)
	<-done

	latest, err := c.GetLatestData("A-B")
	require.NoError(t, err)
	assert.Equal(t, strconv.FormatInt(t2.UnixMilli(), 10), latest.AggregateTime, "LatestData must hold the newer value")
	assert.Equal(t, []string{strconv.FormatInt(t2.UnixMilli(), 10)}, drain(c.OutgoingStream["A-B"]), "stream must not go backwards")
}

// Duplicates of one timestamp (base + sub redis) verified concurrently are delivered once.
func TestConcurrentDuplicateDeliveredOnce(t *testing.T) {
	c, _ := newTestCollector(t, "A-B")
	gate := make(chan struct{})
	ts := time.UnixMilli(1_700_000_000_000)
	f := &fakeOutgoing{gates: map[int64]chan struct{}{ts.UnixMilli(): gate}, entered: make(chan int64, 2)}
	c.toOutgoing = f.convert
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.processIncomingData(ctx, submission("A-B", ts, nil))
		}()
	}
	// both copies are verifying before either delivers (on unfixed code the second is
	// dropped up front, which also yields a single delivery)
	timeout := time.After(time.Second)
	for i := 0; i < 2; i++ {
		select {
		case <-f.entered:
		case <-timeout:
		}
	}
	close(gate)
	wg.Wait()

	assert.Len(t, drain(c.OutgoingStream["A-B"]), 1)
}

// A message failing verification must not block a valid message with the same timestamp.
func TestFailedVerifyDoesNotBlockValidDuplicate(t *testing.T) {
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)
	ts := time.UnixMilli(1_700_000_000_000)
	sig, err := chainutils.MakeValueSignature(100, ts.UnixMilli(), "A-B", pk)
	require.NoError(t, err)
	other, err := crypto.GenerateKey()
	require.NoError(t, err)
	otherSig, err := chainutils.MakeValueSignature(100, ts.UnixMilli(), "A-B", other)
	require.NoError(t, err)

	for name, bad := range map[string][]byte{
		"invalid length":         sig[:10],
		"signer not whitelisted": otherSig,
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := newTestCollector(t, "A-B")
			delete(c.LatestData, "A-B")
			c.FeedHashes["A-B"] = crypto.Keccak256([]byte("A-B"))
			c.CachedWhitelist = []kaiacommon.Address{crypto.PubkeyToAddress(pk.PublicKey)}
			// hold the on-demand whitelist refresh slot: no chain reader in this test
			c.lastOnDemandRefresh.Store(time.Now().Add(time.Hour).UnixNano())
			ctx := context.Background()

			c.processIncomingData(ctx, submission("A-B", ts, bad))
			_, err := c.GetLatestData("A-B")
			require.Error(t, err, "invalid data must not be delivered")

			c.processIncomingData(ctx, submission("A-B", ts, sig))
			latest, err := c.GetLatestData("A-B")
			require.NoError(t, err, "valid duplicate must be delivered")
			assert.Equal(t, strconv.FormatInt(ts.UnixMilli(), 10), latest.AggregateTime)
			assert.Len(t, drain(c.OutgoingStream["A-B"]), 1)
		})
	}
}
