package signer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// stubSeams replaces the chain reads and Slack alert for one test and returns the captured alerts.
func stubSeams(
	t *testing.T,
	addrs func(context.Context, string, string) ([]string, error),
	exp func(context.Context, string, string, string) (*time.Time, error),
) *[]string {
	t.Helper()
	origAddrs, origExp, origAlert := getSignerAddresses, extractExpirationFromContract, slackAlert
	t.Cleanup(func() {
		getSignerAddresses, extractExpirationFromContract, slackAlert = origAddrs, origExp, origAlert
	})

	alerts := []string{}
	getSignerAddresses = addrs
	extractExpirationFromContract = exp
	slackAlert = func(text string) { alerts = append(alerts, text) }
	return &alerts
}

func TestCheckExpirationReadErrorDoesNotPanic(t *testing.T) {
	alerts := stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{"0xsigner1"}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return nil, errors.New("rpc read failed")
		},
	)

	check(context.Background())
	if len(*alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(*alerts))
	}
	msg := (*alerts)[0]
	if !strings.Contains(msg, "0xsigner1") || !strings.Contains(msg, "rpc read failed") {
		t.Fatalf("alert missing signer or failure: %q", msg)
	}
}

func TestCheckNilExpirationWithoutErrorDoesNotPanic(t *testing.T) {
	alerts := stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{"0xsigner1"}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return nil, nil
		},
	)

	check(context.Background())
	if len(*alerts) != 1 || !strings.Contains((*alerts)[0], "0xsigner1") || !strings.Contains((*alerts)[0], "nil expiration") {
		t.Fatalf("expected nil-expiration alert for signer, got %v", *alerts)
	}
}

func TestCheckGetSignerAddressesErrorAlerts(t *testing.T) {
	alerts := stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return nil, errors.New("dial failed")
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			t.Fatal("expiration read must not run when address read fails")
			return nil, nil
		},
	)

	check(context.Background())
	if len(*alerts) != 1 || !strings.Contains((*alerts)[0], "dial failed") {
		t.Fatalf("expected alert with failure, got %v", *alerts)
	}
}

func TestCheckMixedSignersAlertsOnlyProblems(t *testing.T) {
	healthy := time.Now().Add(30 * 24 * time.Hour)
	expiring := time.Now().Add(2 * 24 * time.Hour)
	alerts := stubSeams(t,
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

	check(context.Background())
	if len(*alerts) != 1 {
		t.Fatalf("expected 1 combined alert, got %d", len(*alerts))
	}
	msg := (*alerts)[0]
	if strings.Contains(msg, "0xhealthy") {
		t.Fatalf("healthy signer should not be in alert: %q", msg)
	}
	if !strings.Contains(msg, "0xbroken") || !strings.Contains(msg, "contract read failed") || !strings.Contains(msg, "0xexpiring") {
		t.Fatalf("alert missing expected content: %q", msg)
	}
}

func TestCheckEmptySignerListAlerts(t *testing.T) {
	alerts := stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return []string{}, nil
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			t.Fatal("expiration read must not run for empty signer list")
			return nil, nil
		},
	)

	check(context.Background())
	if len(*alerts) != 1 || !strings.Contains((*alerts)[0], "no signer addresses") {
		t.Fatalf("expected empty-list alert, got %v", *alerts)
	}
}

func TestStartSurvivesReadErrorAndStopsOnCancel(t *testing.T) {
	t.Setenv("JSON_RPC_URL", "http://localhost:0")
	t.Setenv("SUBMISSION_PROXY_CONTRACT", "0x0")
	t.Setenv("SIGNER_CHECK_INTERVAL", "1h")
	alerts := stubSeams(t,
		func(context.Context, string, string) ([]string, error) {
			return nil, errors.New("dial failed")
		},
		func(context.Context, string, string, string) (*time.Time, error) {
			return nil, nil
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Start(ctx); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if len(*alerts) != 1 {
		t.Fatalf("expected startup check alert, got %v", *alerts)
	}
}
