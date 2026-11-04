//nolint:all
package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testBackoff = InitialPriceBackoff{Initial: minInitialPriceBackoff, Max: 2 * minInitialPriceBackoff}

// Regression for #2609: a pool whose first read fails must recover
// without a restart once the read starts succeeding.
func TestFetchInitialPrice_RetriesTransientThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	price, err := FetchInitialPrice(context.Background(), testBackoff, "Test", 7, "test-feed",
		func(ctx context.Context) (*float64, error) {
			if calls.Add(1) <= 3 {
				return nil, errors.New("read tcp: connection reset by peer")
			}
			v := 1.23
			return &v, nil
		},
	)
	require.NoError(t, err)
	require.NotNil(t, price)
	assert.Equal(t, 1.23, *price)
	assert.Equal(t, int32(4), calls.Load())
}

func TestFetchInitialPrice_NilPriceIsRetried(t *testing.T) {
	var calls atomic.Int32
	price, err := FetchInitialPrice(context.Background(), testBackoff, "Test", 7, "test-feed",
		func(ctx context.Context) (*float64, error) {
			if calls.Add(1) == 1 {
				return nil, nil
			}
			v := 2.0
			return &v, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 2.0, *price)
	assert.Equal(t, int32(2), calls.Load())
}

func TestFetchInitialPrice_PermanentErrorReturnsWithoutRetry(t *testing.T) {
	var def struct{ A int }
	jsonErr := json.Unmarshal([]byte(`{"A":"x"}`), &def)
	require.Error(t, jsonErr)

	for name, permErr := range map[string]error{
		"no chain":         errorSentinel.ErrFetcherNoMatchingChainID,
		"wrapped no chain": fmt.Errorf("ctx: %w", errorSentinel.ErrFetcherNoMatchingChainID),
		"bad definition":   fmt.Errorf("%w: %w", errorSentinel.ErrFetcherInvalidDexFetcherDefinition, errors.New("bad poolId")),
		"bad json def":     fmt.Errorf("%w: %w", errorSentinel.ErrFetcherInvalidDexFetcherDefinition, jsonErr),
		"empty address":    errorSentinel.ErrChainEmptyAddressParam,
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			// Long backoff: if the helper retried, the test would time out.
			slow := InitialPriceBackoff{Initial: time.Hour, Max: time.Hour}
			done := make(chan struct{})
			var gotErr error
			go func() {
				_, gotErr = FetchInitialPrice(context.Background(), slow, "Test", 7, "test-feed",
					func(ctx context.Context) (*float64, error) {
						calls.Add(1)
						return nil, permErr
					},
				)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("FetchInitialPrice retried a permanent error")
			}
			assert.ErrorIs(t, gotErr, permErr)
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestFetchInitialPrice_StopsOnContextCancelDuringBackoff(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	slow := InitialPriceBackoff{Initial: time.Hour, Max: time.Hour}

	done := make(chan struct{})
	var gotErr error
	go func() {
		_, gotErr = FetchInitialPrice(ctx, slow, "Test", 7, "test-feed",
			func(ctx context.Context) (*float64, error) {
				calls.Add(1)
				return nil, errors.New("transient")
			},
		)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("FetchInitialPrice did not return after ctx cancel")
	}
	assert.ErrorIs(t, gotErr, context.Canceled)
	assert.Equal(t, int32(1), calls.Load())
}

func TestFetchInitialPrice_BackoffCapped(t *testing.T) {
	var calls atomic.Int32
	start := time.Now()
	_, err := FetchInitialPrice(context.Background(), testBackoff, "Test", 7, "test-feed",
		func(ctx context.Context) (*float64, error) {
			if calls.Add(1) <= 10 {
				return nil, errors.New("transient")
			}
			v := 1.0
			return &v, nil
		},
	)
	require.NoError(t, err)
	// Uncapped doubling from 10ms over 10 failures would be ~10s; capped at
	// 20ms it is ~10+20*9 = 190ms.
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestFetchInitialPrice_ZeroBackoffIsClamped(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := FetchInitialPrice(ctx, InitialPriceBackoff{}, "Test", 7, "test-feed",
		func(ctx context.Context) (*float64, error) {
			calls.Add(1)
			return nil, errors.New("transient")
		},
	)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	// 50ms / 10ms min backoff ≈ 5 calls; a busy loop would be thousands.
	assert.LessOrEqual(t, calls.Load(), int32(10))
}

// A hung read must not stall the retry loop: each attempt gets its own deadline.
func TestFetchInitialPrice_AttemptTimeoutUnblocksHungRead(t *testing.T) {
	var calls atomic.Int32
	b := InitialPriceBackoff{Initial: minInitialPriceBackoff, Max: minInitialPriceBackoff, AttemptTimeout: 20 * time.Millisecond}
	price, err := FetchInitialPrice(context.Background(), b, "Test", 7, "test-feed",
		func(ctx context.Context) (*float64, error) {
			if calls.Add(1) == 1 {
				<-ctx.Done() // hang until the per-attempt deadline
				return nil, ctx.Err()
			}
			v := 3.0
			return &v, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 3.0, *price)
	assert.Equal(t, int32(2), calls.Load())
}

func TestIsPermanentInitialPriceError_TransientErrors(t *testing.T) {
	// Bare json errors can come from RPC response decoding; only the
	// sentinel-wrapped definition unmarshal error is permanent.
	var v any
	syntaxErr := json.Unmarshal([]byte(`{`), &v)
	require.Error(t, syntaxErr)
	for _, err := range []error{
		syntaxErr,
		errors.New("use of closed network connection"),
		context.DeadlineExceeded,
		errorSentinel.ErrFetcherFailedToGetDexResultSlice,
		errorSentinel.ErrFetcherFailedBigIntConvert,
	} {
		assert.False(t, IsPermanentInitialPriceError(err), "%v should be transient", err)
	}
}
