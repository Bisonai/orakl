package metrics

import (
	"context"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaiachain/kaia/common"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartDisabledWhenAddrEmpty(t *testing.T) {
	srv, err := Start("")
	require.NoError(t, err)
	assert.Nil(t, srv)
}

func TestStartServesOnlyMetrics(t *testing.T) {
	srv, ln, err := start("127.0.0.1:0")
	require.NoError(t, err)
	require.NotNil(t, srv)
	t.Cleanup(func() { _ = srv.Close() })
	ObserveSign(ResultOK, "", time.Millisecond)
	addr := ln.String()
	resp, err := http.Get("http://" + addr + "/metrics")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	for _, name := range []string{
		"orakl_delegator_sign_requests_total",
		"orakl_delegator_sign_duration_seconds",
		"orakl_delegator_last_sign_success_timestamp_seconds",
		"orakl_delegator_store_errors_total",
		"orakl_delegator_rpc_errors_total",
		"go_goroutines",
		"process_start_time_seconds",
	} {
		assert.Contains(t, string(body), name)
	}

	for _, path := range []string{"/", "/api/v1/sign/v2", "/api/v1/sign/feePayer"} {
		resp, err := http.Get("http://" + addr + path)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
	}
}

func TestStartBindError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	_, err = Start(ln.Addr().String())
	assert.ErrorContains(t, err, EnvAddr)
}

func TestRunPollRecovers(t *testing.T) {
	assert.NotPanics(t, func() {
		runPoll(context.Background(), func(context.Context) { panic("rpc client bug") })
	})
}

func TestIsLoopback(t *testing.T) {
	assert.True(t, isLoopback(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")}))
	assert.True(t, isLoopback(&net.TCPAddr{IP: net.ParseIP("::1")}))
	assert.False(t, isLoopback(&net.TCPAddr{IP: net.IPv4zero}))
}

type fakeBalance struct {
	peb *big.Int
	err error
}

func (f fakeBalance) BalanceAt(context.Context, common.Address, *big.Int) (*big.Int, error) {
	return f.peb, f.err
}

func TestPollBalance(t *testing.T) {
	addr := func() (common.Address, bool) { return common.HexToAddress("0x1"), true }

	before := testutil.ToFloat64(rpcErrors.WithLabelValues(RPCMethodBalanceAt))
	pollBalance(context.Background(), fakeBalance{err: errors.New("rpc down")}, addr)
	assert.Equal(t, before+1, testutil.ToFloat64(rpcErrors.WithLabelValues(RPCMethodBalanceAt)))

	called := false
	pollBalance(context.Background(), fakeBalance{err: errors.New("unused")}, func() (common.Address, bool) {
		called = true
		return common.Address{}, false
	})
	assert.True(t, called)
	assert.Equal(t, before+1, testutil.ToFloat64(rpcErrors.WithLabelValues(RPCMethodBalanceAt)), "no key loaded: no RPC call")

	peb, _ := new(big.Int).SetString("12500000000000000000", 10)
	pollBalance(context.Background(), fakeBalance{peb: peb}, addr)
	assert.Equal(t, 12.5, testutil.ToFloat64(feePayerBalance))
	assert.Positive(t, testutil.ToFloat64(feePayerBalanceUpdated))
}

func TestPebToKaia(t *testing.T) {
	assert.Equal(t, 0.0, PebToKaia(big.NewInt(0)))
	assert.Equal(t, 1e-18, PebToKaia(big.NewInt(1)))
	assert.Equal(t, 25.0, PebToKaia(new(big.Int).Mul(big.NewInt(25), big.NewInt(1e18))))
}

func TestAllowlistPoller(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	before := testutil.ToFloat64(storeErrors.WithLabelValues(StoreOpAllowlistCount))
	var calls atomic.Int32
	StartAllowlistPoller(ctx, func(context.Context) (int, error) {
		if calls.Add(1) == 1 {
			return 3, nil
		}
		return 0, errors.New("locked")
	}, 10*time.Millisecond)

	assert.Eventually(t, func() bool { return testutil.ToFloat64(allowlistEntries) == 3 }, time.Second, 5*time.Millisecond)
	assert.Eventually(t, func() bool {
		return testutil.ToFloat64(storeErrors.WithLabelValues(StoreOpAllowlistCount)) > before
	}, time.Second, 5*time.Millisecond)
	assert.Equal(t, 3.0, testutil.ToFloat64(allowlistEntries), "a failed count keeps the last value")
}

func TestBalancePollerDisabledWithoutRPC(t *testing.T) {
	before := testutil.ToFloat64(rpcErrors.WithLabelValues(RPCMethodDial))
	StartBalancePoller(context.Background(), "", func() (common.Address, bool) {
		t.Fatal("address must not be read without an RPC URL")
		return common.Address{}, false
	}, time.Millisecond)
	assert.Equal(t, before, testutil.ToFloat64(rpcErrors.WithLabelValues(RPCMethodDial)))
}
