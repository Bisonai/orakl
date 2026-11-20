package ping

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout  = 50 * time.Millisecond
	testInterval = 10 * time.Millisecond
)

// fakePinger replaces probing.Pinger so tests need no ICMP or privileges.
type fakePinger struct {
	err    error
	silent bool // never receives a reply; blocks until ctx is done
	rtt    time.Duration
	recv   int
}

func (f *fakePinger) RunWithContext(ctx context.Context) error {
	if f.silent {
		<-ctx.Done()
		return ctx.Err()
	}
	if f.err != nil {
		return f.err
	}
	f.recv = 1
	return nil
}

func (f *fakePinger) Statistics() *probing.Statistics {
	return &probing.Statistics{PacketsRecv: f.recv, AvgRtt: f.rtt}
}

// newTestApp builds an App whose pingers come from mk, with short timings.
func newTestApp(t *testing.T, endpoints []string, mk func() *fakePinger) (*App, *atomic.Int64) {
	t.Helper()
	app, err := New(WithEndpoints(endpoints))
	require.NoError(t, err)
	app.timeout = testTimeout
	app.interval = testInterval
	runs := &atomic.Int64{}
	app.newPinger = func(string, time.Duration) (PingerInterface, error) {
		runs.Add(1)
		return mk(), nil
	}
	return app, runs
}

// startWithDeadline runs Start and fails the test if it does not return in time.
func startWithDeadline(t *testing.T, app *App, ctx context.Context, deadline time.Duration) time.Duration {
	t.Helper()
	done := make(chan struct{})
	begin := time.Now()
	go func() {
		defer close(done)
		app.Start(ctx)
	}()
	select {
	case <-done:
		return time.Since(begin)
	case <-time.After(deadline):
		t.Fatalf("Start did not return within %s", deadline)
		return 0
	}
}

func TestApp_Start_SuccessfulPing(t *testing.T) {
	app, _ := newTestApp(t, []string{"8.8.8.8"}, func() *fakePinger {
		return &fakePinger{rtt: 100 * time.Millisecond}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	startWithDeadline(t, app, ctx, 2*time.Second)

	assert.Equal(t, 0, app.FailCount["8.8.8.8"])
}

func TestApp_Start_FailedPing(t *testing.T) {
	app, _ := newTestApp(t, []string{"8.8.8.8"}, func() *fakePinger {
		return &fakePinger{err: errors.New("ping failed")}
	})
	app.interval = time.Hour // exactly one attempt

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	startWithDeadline(t, app, ctx, 2*time.Second)

	assert.Equal(t, 1, app.FailCount["8.8.8.8"])
}

func TestApp_Start_ShutdownOnAllFailures(t *testing.T) {
	endpoints := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}
	app, _ := newTestApp(t, endpoints, func() *fakePinger {
		return &fakePinger{err: errors.New("ping failed")}
	})

	startWithDeadline(t, app, context.Background(), 2*time.Second)

	for _, addr := range endpoints {
		assert.GreaterOrEqual(t, app.FailCount[addr], DefaultMaxFails)
	}
}

// A reply that never arrives is reported as a failure within the timeout.
func TestApp_RunOnce_NoReplyIsFailure(t *testing.T) {
	app, _ := newTestApp(t, []string{"8.8.8.8"}, func() *fakePinger {
		return &fakePinger{silent: true}
	})

	begin := time.Now()
	result, ok := app.runOnce(context.Background(), "8.8.8.8")
	elapsed := time.Since(begin)

	assert.True(t, ok)
	assert.False(t, result.Success)
	assert.Less(t, elapsed, testTimeout+time.Second)
}

// Run returns nil but no packet was received (pro-bing's own Timeout fired).
func TestApp_RunOnce_ZeroPacketsIsFailure(t *testing.T) {
	app, _ := newTestApp(t, []string{"8.8.8.8"}, nil)
	app.newPinger = func(string, time.Duration) (PingerInterface, error) {
		return zeroRecvPinger{}, nil
	}

	result, ok := app.runOnce(context.Background(), "8.8.8.8")
	assert.True(t, ok)
	assert.False(t, result.Success)
}

type zeroRecvPinger struct{}

func (zeroRecvPinger) RunWithContext(context.Context) error { return nil }
func (zeroRecvPinger) Statistics() *probing.Statistics {
	return &probing.Statistics{PacketsSent: 1}
}

// All endpoints silent: Start returns within ~MaxFails x (timeout + interval).
func TestApp_Start_AllSilentShutsDown(t *testing.T) {
	endpoints := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}
	app, _ := newTestApp(t, endpoints, func() *fakePinger {
		return &fakePinger{silent: true}
	})

	limit := DefaultMaxFails*(testTimeout+testInterval) + time.Second
	elapsed := startWithDeadline(t, app, context.Background(), 5*time.Second)

	assert.Less(t, elapsed, limit)
	for _, addr := range endpoints {
		assert.GreaterOrEqual(t, app.FailCount[addr], DefaultMaxFails)
	}
}

// Each attempt uses a fresh pinger (probing.Pinger is single-use).
func TestApp_Start_FreshPingerPerAttempt(t *testing.T) {
	app, runs := newTestApp(t, []string{"8.8.8.8"}, func() *fakePinger {
		return &fakePinger{rtt: time.Millisecond}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	startWithDeadline(t, app, ctx, 2*time.Second)

	assert.Greater(t, runs.Load(), int64(1))
}

func TestApp_ProcessResult_FirstSampleNotFailure(t *testing.T) {
	app, err := New(WithEndpoints([]string{"8.8.8.8"}))
	require.NoError(t, err)

	app.processResult(PingResult{Address: "8.8.8.8", Success: true, Delay: 20 * time.Millisecond})
	assert.Equal(t, 0, app.FailCount["8.8.8.8"])
	assert.Equal(t, float64(20*time.Millisecond), app.RTTAvg["8.8.8.8"])

	// Within threshold: still OK.
	app.processResult(PingResult{Address: "8.8.8.8", Success: true, Delay: 22 * time.Millisecond})
	assert.Equal(t, 0, app.FailCount["8.8.8.8"])

	// Well above threshold: counted.
	app.processResult(PingResult{Address: "8.8.8.8", Success: true, Delay: 200 * time.Millisecond})
	assert.Equal(t, 1, app.FailCount["8.8.8.8"])
}

// Sub-millisecond jitter is tolerated, but RTTs are compared as durations,
// not truncated to 0 ms.
func TestApp_ProcessResult_SubMillisecond(t *testing.T) {
	app, err := New(WithEndpoints([]string{"8.8.8.8"}))
	require.NoError(t, err)

	app.processResult(PingResult{Address: "8.8.8.8", Success: true, Delay: 200 * time.Microsecond})
	app.processResult(PingResult{Address: "8.8.8.8", Success: true, Delay: 900 * time.Microsecond})
	assert.Equal(t, 0, app.FailCount["8.8.8.8"])

	app.processResult(PingResult{Address: "8.8.8.8", Success: true, Delay: 1900 * time.Microsecond})
	assert.Equal(t, 1, app.FailCount["8.8.8.8"])
}

// ctx cancel: no panic, Start returns and all ping goroutines end.
func TestApp_Start_CancelNoPanic(t *testing.T) {
	endpoints := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}
	app, _ := newTestApp(t, endpoints, func() *fakePinger {
		return &fakePinger{rtt: time.Millisecond}
	})
	app.interval = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	startWithDeadline(t, app, ctx, 2*time.Second)

	// Start waits for its goroutines, so nothing is sent afterwards.
	n := len(app.ResultsBuffer)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, n, len(app.ResultsBuffer))
}
