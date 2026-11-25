package helper

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	metricnames "bisonai.com/orakl/node/pkg/metrics/names"
	signermetrics "bisonai.com/orakl/node/pkg/metrics/signer"
	"github.com/kaiachain/kaia/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Failed-rotation backoff (issue #2965): a rotation that cannot land must not keep the node
// from signing with its still-whitelisted key, and must be retried with exponential backoff.

// hookChain runs onUpdate at the start of every UpdateOracle call (before any broadcast).
type hookChain struct {
	*fakeChain
	onUpdate func()
}

func (c *hookChain) UpdateOracle(ctx context.Context, from string, newAddr common.Address, onBroadcast func()) error {
	c.onUpdate()
	return c.fakeChain.UpdateOracle(ctx, from, newAddr, onBroadcast)
}

func (f *fakeChain) calls() int { f.mu.Lock(); defer f.mu.Unlock(); return f.updateCalls }

func rotationFailureCount(t *testing.T, reason string) float64 {
	t.Helper()
	signermetrics.Register()
	mfs, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != metricnames.SignerRotationFailures {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == metricnames.ReasonLabel && l.GetValue() == reason {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	t.Fatalf("series %s{reason=%q} not exported", metricnames.SignerRotationFailures, reason)
	return 0
}

// nearExpirySigner returns a signer whose active key is whitelisted but within the renew
// threshold (so every allowRotate reconcile wants to rotate), with a fake backoff clock.
func nearExpirySigner(t *testing.T, chain oracleChain, fc *fakeChain) (*Signer, *fakeStore, common.Address, *time.Time) {
	t.Helper()
	aHex, aAddr, aPK := genKey(t)
	store := newFakeStore()
	store.addKey(aHex, aAddr, "active")
	fc.setExp(aAddr, 3*24*time.Hour)

	s := newScenarioSigner(chain, store)
	s.PK, s.activeAddr, s.usable, s.cachedExpiration = aPK, aAddr, true, time.Now().Add(3*24*time.Hour)
	s.lastConfirmedAt = time.Now()
	clk := time.Now()
	s.now = func() time.Time { return clk }
	return s, store, aAddr, &clk
}

func sign(s *Signer) error {
	_, err := s.MakeGlobalAggregateProof(1, time.Now(), "X")
	return err
}

// UpdateOracle fails before broadcast: the node keeps signing with the old key the whole time
// (also during the attempt), keeps the pending key, and retries with growing backoff.
func TestRotationNotSentKeepsSigningAndBacksOff(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator 403, direct fallback: suggest gas price failed")
	var duringErrs []error
	hc := &hookChain{fakeChain: fc}
	s, store, aAddr, clk := nearExpirySigner(t, hc, fc)
	hc.onUpdate = func() { duringErrs = append(duringErrs, sign(s)) }
	before := rotationFailureCount(t, signermetrics.ReasonNotSent)

	require.NoError(t, s.reconcile(ctx, true))
	require.Equal(t, 1, fc.calls())
	require.NoError(t, duringErrs[0], "must keep signing while updateOracle is built/sent")
	require.NoError(t, sign(s), "must keep signing after a not-sent rotation")
	assert.False(t, s.Status().Rotating)
	assert.Equal(t, aAddr, s.activeAddr)
	assert.NotNil(t, store.findState("pending"), "pending key kept for reuse")
	assert.Equal(t, before+1, rotationFailureCount(t, signermetrics.ReasonNotSent))

	// retries only after 5m, 10m, 20m, 40m, then every 1h
	for i, wait := range []time.Duration{5 * time.Minute, 10 * time.Minute, 20 * time.Minute, 40 * time.Minute, time.Hour, time.Hour} {
		*clk = clk.Add(wait - time.Second)
		require.NoError(t, s.reconcile(ctx, true))
		require.Equal(t, i+1, fc.calls(), "retried before backoff %s elapsed", wait)
		*clk = clk.Add(time.Second)
		require.NoError(t, s.reconcile(ctx, true))
		require.Equal(t, i+2, fc.calls(), "not retried after backoff %s", wait)
	}
	assert.Equal(t, 2, store.count(), "retries reuse the one pending key")
}

// Duty cycle: liveness ticks every 30s for 12h while rotation cannot be sent. Every tick must
// still sign (master: rotation blocked signing on every tick), with few rotation attempts.
func TestRotationNotSentDutyCycle(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator down")
	var duringFailures int
	hc := &hookChain{fakeChain: fc}
	s, _, _, clk := nearExpirySigner(t, hc, fc)
	hc.onUpdate = func() {
		if sign(s) != nil {
			duringFailures++
		}
	}

	ticks, signed := 0, 0
	for d := time.Duration(0); d < 12*time.Hour; d += s.livenessInterval {
		*clk = clk.Add(s.livenessInterval)
		require.NoError(t, s.reconcile(ctx, true))
		ticks++
		if sign(s) == nil {
			signed++
		}
	}
	assert.Equal(t, ticks, signed, "every tick must sign")
	assert.Zero(t, duringFailures, "no refusals during rotation attempts")
	// attempts at 0, 5m, 15m, 35m, 75m, then hourly: 15 within 12h
	assert.LessOrEqual(t, fc.calls(), 15)
	assert.GreaterOrEqual(t, fc.calls(), 13)
}

// The node refuses the sent tx (delegator down, old key unfunded): treated as not sent, no
// verify poll, signing resumes at once, and the attempt backs off.
func TestRotationTxpoolRejectIsNotSent(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	s, store, aAddr, clk := nearExpirySigner(t, fc, fc)
	fc.updateOracle = func(*fakeChain, string, common.Address) error {
		return errors.New("insufficient funds of the sender for value ")
	}
	before := rotationFailureCount(t, signermetrics.ReasonNotSent)
	start := time.Now()
	s.verifyPollInterval = time.Second // a poll would take >= 3s

	require.NoError(t, s.reconcile(ctx, true))
	assert.Less(t, time.Since(start), time.Second, "must not poll for a rejected tx")
	require.NoError(t, sign(s))
	assert.False(t, s.Status().Rotating)
	assert.Equal(t, aAddr, s.activeAddr)
	assert.NotNil(t, store.findState("pending"))
	assert.Equal(t, before+1, rotationFailureCount(t, signermetrics.ReasonNotSent))

	*clk = clk.Add(5*time.Minute - time.Second)
	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, 1, fc.calls(), "backs off")
}

// Broadcast then not observed on-chain: unchanged behaviour (refuse during the in-flight window,
// resume with the old key after), and the failure backs off too.
func TestRotationNotLandedBacksOff(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	var during error
	s, store, aAddr, clk := nearExpirySigner(t, fc, fc)
	fc.updateOracle = func(*fakeChain, string, common.Address) error {
		during = sign(s)
		return errors.New("wait mined: timeout")
	}
	before := rotationFailureCount(t, signermetrics.ReasonNotLanded)

	require.NoError(t, s.reconcile(ctx, true))
	require.ErrorIs(t, during, errorSentinel.ErrChainSignerNoWhitelistedKey, "must refuse once updateOracle is broadcast")
	require.NoError(t, sign(s), "must resume with the still-whitelisted old key")
	assert.Equal(t, aAddr, s.activeAddr)
	assert.NotNil(t, store.findState("pending"))
	assert.Equal(t, before+1, rotationFailureCount(t, signermetrics.ReasonNotLanded))

	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, 1, fc.calls(), "no retry before backoff")
	*clk = clk.Add(5 * time.Minute)
	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, 2, fc.calls(), "retry after backoff")
}

// A successful rotation after failures resets the backoff.
func TestRotationSuccessResetsBackoff(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator down")
	s, _, aAddr, clk := nearExpirySigner(t, fc, fc)

	require.NoError(t, s.reconcile(ctx, true))
	*clk = clk.Add(5 * time.Minute)
	require.NoError(t, s.reconcile(ctx, true))
	require.Equal(t, 2, s.rotationFailures)

	fc.mu.Lock()
	fc.notSentErr = nil // delegator back: updateOracle mines
	fc.mu.Unlock()
	*clk = clk.Add(10 * time.Minute)
	require.NoError(t, s.reconcile(ctx, true))
	require.NotEqual(t, aAddr, s.activeAddr, "expected rotation")
	require.NoError(t, sign(s))
	assert.Zero(t, s.rotationFailures)
	assert.True(t, s.nextRotationAt.IsZero())
}

// The operator trigger (admin renew) retries at once, without waiting out the backoff.
func TestCheckAndUpdateSignerPKSkipsBackoffWait(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator down")
	s, _, _, _ := nearExpirySigner(t, fc, fc)

	require.NoError(t, s.reconcile(ctx, true))
	require.NoError(t, s.reconcile(ctx, true))
	require.Equal(t, 1, fc.calls())

	require.NoError(t, s.CheckAndUpdateSignerPK(ctx))
	assert.Equal(t, 2, fc.calls())
	assert.Equal(t, 2, s.rotationFailures, "failure count kept")
}

// An operator trigger that arrives while a reconcile holds the lock is not lost: the running
// attempt's failure schedules a backoff, but the next reconcile still retries at once.
func TestCheckAndUpdateSignerPKDuringRunningReconcile(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator down")
	hc := &hookChain{fakeChain: fc}
	s, _, _, _ := nearExpirySigner(t, hc, fc)
	hc.onUpdate = func() {
		if fc.calls() == 0 { // inside the first attempt, rotateMu held
			require.NoError(t, s.CheckAndUpdateSignerPK(ctx))
		}
	}

	require.NoError(t, s.reconcile(ctx, true))
	require.Equal(t, 1, fc.calls())
	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, 2, fc.calls(), "forced retry lost")
	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, 2, fc.calls(), "force is one-shot")
}

// The backoff never schedules the retry past the last moment the active key can send it.
func TestRotationBackoffCappedByExpiry(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator down")
	s, _, aAddr, clk := nearExpirySigner(t, fc, fc)
	fc.setExp(aAddr, 20*time.Minute)
	s.rotationFailures = 5 // next delay would be 1h

	require.NoError(t, s.reconcile(ctx, true))
	lastChance := s.cachedExpiration.Add(-s.skewMargin - time.Duration(s.verifyPollMax)*s.verifyPollInterval)
	assert.Equal(t, lastChance, s.nextRotationAt)
	assert.True(t, s.nextRotationAt.Before(clk.Add(20*time.Minute)))
}

// A Signer built without backoff config still backs off (defaults).
func TestRotationBackoffZeroConfigUsesDefaults(t *testing.T) {
	ctx := context.Background()
	fc := newFakeChain()
	fc.notSentErr = errors.New("delegator down")
	s, _, _, clk := nearExpirySigner(t, fc, fc)
	s.rotationBackoffBase, s.rotationBackoffMax = 0, 0

	require.NoError(t, s.reconcile(ctx, true))
	assert.Equal(t, clk.Add(DefaultSignerRotationBackoffBase), s.nextRotationAt)
}

// beforeSend runs once right before the tx is sent, and not when building it fails.
func TestSubmitDelegatedBeforeSend(t *testing.T) {
	ctx := context.Background()

	h, client := newNonceTestHelper(t)
	var sentAtHook []int
	h.beforeSend = func() { sentAtHook = append(sentAtHook, len(client.sent)) }
	_ = h.submitDelegatedFallbackDirect(ctx, "rotate", "0x1234", testSubmitFunc, big.NewInt(1))
	assert.Equal(t, []int{0}, sentAtHook, "called once, before SendTransaction")
	assert.Len(t, client.sent, 1)

	h2, client2 := newNonceTestHelper(t, 1) // fee-delegated build fails
	called := false
	h2.beforeSend = func() { called = true }
	require.ErrorIs(t, h2.submitDelegatedFallbackDirect(ctx, "rotate", "0x1234", testSubmitFunc, big.NewInt(1)), errFakeRPC)
	assert.False(t, called)
	assert.Empty(t, client2.sent)
}
