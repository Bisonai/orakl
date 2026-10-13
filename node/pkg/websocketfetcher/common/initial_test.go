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

var testBackoff = InitialPriceBackoff{Initial: time.Millisecond, Max: 4 * time.Millisecond}

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
	var v any
	syntaxErr := json.Unmarshal([]byte(`{`), &v)
	require.Error(t, syntaxErr)

	for name, permErr := range map[string]error{
		"no chain":         errorSentinel.ErrFetcherNoMatchingChainID,
		"wrapped no chain": fmt.Errorf("ctx: %w", errorSentinel.ErrFetcherNoMatchingChainID),
		"bad definition":   fmt.Errorf("%w: %w", errorSentinel.ErrFetcherInvalidDexFetcherDefinition, errors.New("bad poolId")),
		"json type":        jsonErr,
		"json syntax":      syntaxErr,
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
	// Uncapped doubling from 1ms over 10 failures would be ~1s; capped at
	// 4ms it is ~1+2+4*8 = 35ms.
	assert.Less(t, time.Since(start), 500*time.Millisecond)
}

func TestIsPermanentInitialPriceError_TransientErrors(t *testing.T) {
	for _, err := range []error{
		errors.New("use of closed network connection"),
		context.DeadlineExceeded,
		errorSentinel.ErrFetcherFailedToGetDexResultSlice,
		errorSentinel.ErrFetcherFailedBigIntConvert,
	} {
		assert.False(t, IsPermanentInitialPriceError(err), "%v should be transient", err)
	}
}
