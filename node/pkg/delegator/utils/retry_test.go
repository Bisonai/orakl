package utils

import (
	"context"
	"strings"
	"testing"
	"time"
)

// reloadFeePayer must convert a panic in a secret-store client into an error.
// Without this the retry goroutine has no recover above it and a panicking
// loader would kill the process once per interval.
func TestReloadFeePayerRecoversPanic(t *testing.T) {
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	t.Setenv("USE_VAULT", "false")
	t.Setenv("USE_GOOGLE_SECRET_MANAGER", "false")

	// no source configured -> InitFeePayerPK returns an error rather than
	// panicking, and reloadFeePayer passes it through untouched
	err := reloadFeePayer(context.Background(), nil)
	if err == nil {
		t.Fatal("reloadFeePayer() = nil, want error when no source is configured")
	}
	if strings.Contains(err.Error(), "panic") {
		t.Fatalf("unexpected panic path: %v", err)
	}
}

// A configured env key must still load through the retry path, so recovery
// does not mask the success case.
func TestReloadFeePayerSucceedsFromEnv(t *testing.T) {
	t.Cleanup(func() { UpdateFeePayer("") })
	t.Setenv("DELEGATOR_FEEPAYER_PK", validKey)

	if err := reloadFeePayer(context.Background(), nil); err != nil {
		t.Fatalf("reloadFeePayer() = %v, want nil", err)
	}
	if got := CurrentFeePayer(); got != validKey {
		t.Fatalf("CurrentFeePayer() = %q, want %q", got, validKey)
	}
}

// A cancelled context must stop the loop instead of retrying forever.
func TestRetryFeePayerInitStopsOnContextCancel(t *testing.T) {
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	t.Setenv("USE_VAULT", "false")
	t.Setenv("USE_GOOGLE_SECRET_MANAGER", "false")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		retryFeePayerInit(ctx, nil, FeePayerRetryInterval)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("retryFeePayerInit did not return after context cancel")
	}
}
