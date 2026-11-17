package signer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// stubSeams replaces the chain reads for one test and resets the exported metrics.
func stubSeams(
	t *testing.T,
	addrs func(context.Context, string, string) ([]string, error),
	exp func(context.Context, string, string, string) (*time.Time, error),
) {
	t.Helper()
	origAddrs, origExp := getSignerAddresses, extractExpirationFromContract
	t.Cleanup(func() {
		getSignerAddresses, extractExpirationFromContract = origAddrs, origExp
	})

	getSignerAddresses = addrs
	extractExpirationFromContract = exp
	expiryGauge.Reset()
	countGauge.Set(0)
	lastSuccessGauge.Set(0)
}

func errorsDelta(t *testing.T, f func()) float64 {
	t.Helper()
	before := testutil.ToFloat64(errorsCounter)
	f()
	return testutil.ToFloat64(errorsCounter) - before
}

func TestCheckExpirationReadErrorDoesNotPanic(t *testing.T) {
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{"0xsigner1"}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return nil, errors.New("rpc read failed")
		},
	)

	if d := errorsDelta(t, func() { check(context.Background()) }); d != 1 {
		t.Fatalf("expected 1 error, got %v", d)
	}
	if n := testutil.CollectAndCount(expiryGauge); n != 0 {
		t.Fatalf("expected no expiry series, got %d", n)
	}
	if v := testutil.ToFloat64(lastSuccessGauge); v != 0 {
		t.Fatalf("last success must not be set on failure, got %v", v)
	}
}

func TestCheckNilExpirationWithoutErrorDoesNotPanic(t *testing.T) {
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{"0xsigner1"}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return nil, nil
		},
	)

	if d := errorsDelta(t, func() { check(context.Background()) }); d != 1 {
		t.Fatalf("expected 1 error for nil expiration, got %v", d)
	}
}

func TestCheckGetSignerAddressesErrorCounts(t *testing.T) {
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return nil, errors.New("dial failed")
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			t.Fatal("expiration read must not run when address read fails")
			return nil, nil
		},
	)

	if d := errorsDelta(t, func() { check(context.Background()) }); d != 1 {
		t.Fatalf("expected 1 error, got %v", d)
	}
}

func TestCheckMixedSignersExportsHealthyOnes(t *testing.T) {
	healthy := time.Now().Add(30 * 24 * time.Hour)
	expiring := time.Now().Add(2 * 24 * time.Hour)
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{"0xhealthy", "0xbroken", "0xexpiring"}, nil
		},
		func(_ context.Context, _ string, _ string, signer string) (*time.Time, error) {
			switch signer {
			case "0xhealthy":
				return &healthy, nil
			case "0xexpiring":
				return &expiring, nil
			default:
				return nil, errors.New("contract read failed")
			}
		},
	)

	if d := errorsDelta(t, func() { check(context.Background()) }); d != 1 {
		t.Fatalf("expected 1 error, got %v", d)
	}
	if v := testutil.ToFloat64(countGauge); v != 3 {
		t.Fatalf("expected count 3, got %v", v)
	}
	if v := testutil.ToFloat64(expiryGauge.WithLabelValues("0xhealthy")); v != float64(healthy.Unix()) {
		t.Fatalf("healthy expiry: got %v", v)
	}
	if v := testutil.ToFloat64(expiryGauge.WithLabelValues("0xexpiring")); v != float64(expiring.Unix()) {
		t.Fatalf("expiring expiry: got %v", v)
	}
	if v := testutil.ToFloat64(lastSuccessGauge); v != 0 {
		t.Fatalf("last success must not be set when a read failed, got %v", v)
	}
}

func TestCheckAllReadsOkSetsLastSuccess(t *testing.T) {
	exp := time.Now().Add(30 * 24 * time.Hour)
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{"0xa"}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return &exp, nil
		},
	)

	check(context.Background())
	if v := testutil.ToFloat64(lastSuccessGauge); v == 0 {
		t.Fatal("expected last success to be set")
	}
}

func TestCheckDropsDeregisteredSigner(t *testing.T) {
	exp := time.Now().Add(30 * 24 * time.Hour)
	signers := []string{"0xa", "0xb"}
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return signers, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return &exp, nil
		},
	)

	check(context.Background())
	signers = []string{"0xa"}
	check(context.Background())
	if n := testutil.CollectAndCount(expiryGauge); n != 1 {
		t.Fatalf("expected 1 expiry series after deregistration, got %d", n)
	}
}

func TestCheckEmptySignerListExportsZero(t *testing.T) {
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			t.Fatal("expiration read must not run for empty signer list")
			return nil, nil
		},
	)

	countGauge.Set(5)
	check(context.Background())
	if v := testutil.ToFloat64(countGauge); v != 0 {
		t.Fatalf("expected count 0, got %v", v)
	}
}

func TestStartSurvivesReadErrorAndStopsOnCancel(t *testing.T) {
	t.Setenv("JSON_RPC_URL", "http://localhost:0")
	t.Setenv("SUBMISSION_PROXY_CONTRACT", "0x0")
	t.Setenv("SIGNER_CHECK_INTERVAL", "1h")
	stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return nil, errors.New("dial failed")
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return nil, nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := errorsDelta(t, func() {
		if err := Start(ctx); err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	})
	if d != 1 {
		t.Fatalf("expected startup check error, got %v", d)
	}
}

func TestSetUpNonPositiveIntervalUsesDefault(t *testing.T) {
	t.Setenv("JSON_RPC_URL", "http://localhost:0")
	t.Setenv("SUBMISSION_PROXY_CONTRACT", "0x0")
	for _, v := range []string{"0s", "-1h"} {
		t.Setenv("SIGNER_CHECK_INTERVAL", v)
		if err := setUp(context.Background()); err != nil {
			t.Fatalf("setUp(%s): %v", v, err)
		}
		if signerCheckInterval != DEFAULT_SIGNER_CHECK_INTERVAL_HOUR*time.Hour {
			t.Fatalf("interval %s: got %s, want default", v, signerCheckInterval)
		}
	}
}
