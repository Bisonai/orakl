package utils

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// noFeePayerSource points InitFeePayerPK at nothing, so it fails without
// touching a real secret store.
func noFeePayerSource(t *testing.T) {
	t.Helper()
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	t.Setenv("USE_VAULT", "false")
	t.Setenv("USE_GOOGLE_SECRET_MANAGER", "false")
}

// The retry goroutine has no fiber recover middleware above it, so a panicking
// secret-store client would kill the process once per interval.
func TestReloadFeePayerRecoversPanic(t *testing.T) {
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	t.Setenv("USE_VAULT", "true")
	t.Setenv("USE_GOOGLE_SECRET_MANAGER", "false")

	orig := loadFeePayerFromVault
	t.Cleanup(func() { loadFeePayerFromVault = orig; UpdateFeePayer("") })
	loadFeePayerFromVault = func(context.Context) (string, error) {
		panic("vault client blew up")
	}

	err := reloadFeePayer(context.Background(), nil)
	if err == nil {
		t.Fatal("reloadFeePayer() = nil, want error when the loader panics")
	}
	if !strings.Contains(err.Error(), "panic while loading fee payer") {
		t.Fatalf("reloadFeePayer() = %v, want the panic surfaced as an error", err)
	}
	if !strings.Contains(err.Error(), "vault client blew up") {
		t.Fatalf("reloadFeePayer() = %v, want the panic value preserved", err)
	}
}

// A loader returning an ordinary error must pass through untouched, not be
// dressed up as a panic.
func TestReloadFeePayerPassesThroughError(t *testing.T) {
	t.Setenv("DELEGATOR_FEEPAYER_PK", "")
	t.Setenv("USE_VAULT", "true")
	t.Setenv("USE_GOOGLE_SECRET_MANAGER", "false")

	orig := loadFeePayerFromVault
	sentinel := errors.New("vault unreachable")
	t.Cleanup(func() { loadFeePayerFromVault = orig; UpdateFeePayer("") })
	loadFeePayerFromVault = func(context.Context) (string, error) { return "", sentinel }

	err := reloadFeePayer(context.Background(), nil)
	if !errors.Is(err, sentinel) {
		t.Fatalf("reloadFeePayer() = %v, want %v", err, sentinel)
	}
}

// Recovery must not mask the success path.
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

// Shutting the app down must cancel the retry loop. Without this each Setup
// leaks a goroutine that nothing can stop.
func TestFeePayerRetryStopsOnAppShutdown(t *testing.T) {
	noFeePayerSource(t)
	t.Cleanup(func() { UpdateFeePayer("") })

	app := fiber.New()
	ctx := startFeePayerRetry(app, nil, FeePayerRetryInterval)

	select {
	case <-ctx.Done():
		t.Fatal("retry context cancelled before shutdown")
	default:
	}

	if err := app.Shutdown(); err != nil {
		t.Fatalf("app.Shutdown() = %v, want nil", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("app shutdown did not cancel the retry context")
	}
}

// A cancelled context must stop the loop rather than retrying forever.
func TestRetryFeePayerInitStopsOnContextCancel(t *testing.T) {
	noFeePayerSource(t)

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
