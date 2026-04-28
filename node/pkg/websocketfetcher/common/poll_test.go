//nolint:all
package common

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Regression test for the lifecycle bug where polling fired exactly once
// because the surrounding run() invoked defer cancel() after subscribeEvent
// returned.  In the helper form, the loop's lifetime is bound only to its
// own ctx — fetchPrice should be called repeatedly until the test cancels.
func TestHeartbeatPoll_FiresOnEveryTick(t *testing.T) {
	var calls atomic.Int32
	emits := make(chan *FeedData, 16)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		HeartbeatPoll(ctx, 10*time.Millisecond, "Test", 7, "test-feed",
			func(ctx context.Context) (*float64, error) {
				calls.Add(1)
				v := 1.23
				return &v, nil
			},
			func(fd *FeedData) { emits <- fd },
		)
		close(done)
	}()

	// Allow ~5 ticks worth of time, then cancel.
	time.Sleep(55 * time.Millisecond)
	cancel()
	<-done

	got := calls.Load()
	assert.GreaterOrEqual(t, got, int32(3), "expected polling to fire on every ticker tick")
	assert.LessOrEqual(t, len(emits), int(got), "emits should never exceed calls")
}

func TestHeartbeatPoll_ContinuesAfterFetchError(t *testing.T) {
	var calls atomic.Int32
	emits := make(chan *FeedData, 16)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		HeartbeatPoll(ctx, 10*time.Millisecond, "Test", 7, "test-feed",
			func(ctx context.Context) (*float64, error) {
				n := calls.Add(1)
				// First two calls fail, then succeed forever.
				if n <= 2 {
					return nil, errors.New("transient")
				}
				v := 9.99
				return &v, nil
			},
			func(fd *FeedData) { emits <- fd },
		)
		close(done)
	}()

	time.Sleep(75 * time.Millisecond)
	cancel()
	<-done

	got := calls.Load()
	assert.GreaterOrEqual(t, got, int32(5), "loop should keep ticking through errors")
	assert.GreaterOrEqual(t, len(emits), 1, "successful calls should still emit")
}

func TestHeartbeatPoll_StopsOnContextCancel(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		HeartbeatPoll(ctx, 5*time.Millisecond, "Test", 7, "test-feed",
			func(ctx context.Context) (*float64, error) {
				calls.Add(1)
				v := 1.0
				return &v, nil
			},
			func(*FeedData) {},
		)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("HeartbeatPoll did not return after ctx cancel")
	}

	before := calls.Load()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, before, calls.Load(), "no further calls after cancel")
}

func TestHeartbeatPoll_NilPriceSkipsEmit(t *testing.T) {
	emitCount := 0
	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		HeartbeatPoll(ctx, 5*time.Millisecond, "Test", 7, "test-feed",
			func(ctx context.Context) (*float64, error) {
				return nil, nil // valid call, no price
			},
			func(*FeedData) {
				mu.Lock()
				emitCount++
				mu.Unlock()
			},
		)
		close(done)
	}()

	time.Sleep(40 * time.Millisecond)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 0, emitCount, "nil price must not be emitted")
}

func TestGetDexPollInterval_DefaultWhenUnset(t *testing.T) {
	t.Setenv("DEX_POLL_INTERVAL", "")
	assert.Equal(t, DefaultDexPollInterval, GetDexPollInterval())
}

func TestGetDexPollInterval_DefaultWhenUnparseable(t *testing.T) {
	t.Setenv("DEX_POLL_INTERVAL", "not-a-duration")
	assert.Equal(t, DefaultDexPollInterval, GetDexPollInterval())
}

func TestGetDexPollInterval_DefaultWhenZeroOrNegative(t *testing.T) {
	for _, v := range []string{"0s", "-5s", "0"} {
		t.Setenv("DEX_POLL_INTERVAL", v)
		assert.Equal(t, DefaultDexPollInterval, GetDexPollInterval(), "input %q should fall back", v)
	}
}

func TestGetDexPollInterval_ParsesValidDuration(t *testing.T) {
	t.Setenv("DEX_POLL_INTERVAL", "15s")
	assert.Equal(t, 15*time.Second, GetDexPollInterval())
}

// Sanity that we're using the same env name everyone else thinks we're using.
func TestGetDexPollInterval_EnvName(t *testing.T) {
	const envKey = "DEX_POLL_INTERVAL"
	old := os.Getenv(envKey)
	defer os.Setenv(envKey, old)
	os.Setenv(envKey, "42s")
	assert.Equal(t, 42*time.Second, GetDexPollInterval())
}
