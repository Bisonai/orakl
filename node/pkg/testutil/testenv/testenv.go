// Package testenv lets tests run with no external services and no secrets.
//
// Redis: when REDIS_HOST is unset, StartRedis (for TestMain) and Redis (per
// test) start an in-process miniredis server and point REDIS_HOST/REDIS_PORT at
// it. When REDIS_HOST is already set (CI service container, local redis), both
// do nothing, so behaviour against a real Redis is unchanged.
//
// The in-process server is shared by the whole test binary and keeps one
// address for its lifetime, so the lazy client in pkg/db stays valid across
// tests. This package does not import pkg/db: pkg/db's internal tests must be
// able to use it without an import cycle. Callers close the shared client
// themselves (db.CloseRedis) before calling the StartRedis cleanup.
//
// Postgres and other env-dependent resources: Postgres and Env skip the test
// with a clear reason when the required variables are missing.
//
// Build tag convention: tests that need public networks (RPC endpoints, CEX
// websockets) or funded keys belong behind the `integration` build tag
// (`//go:build integration`) and run with `go test -tags integration ./...`.
// Plain `go test ./...` must pass on any machine.
package testenv

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

const (
	redisHostEnv = "REDIS_HOST"
	redisPortEnv = "REDIS_PORT"
	// DatabaseURLEnv is the Postgres connection string read by pkg/db.
	DatabaseURLEnv = "DATABASE_URL"
)

var (
	redisMu     sync.Mutex
	redisServer *miniredis.Miniredis
)

// startShared starts the shared miniredis server if it is not running and
// returns its host and port.
func startShared() (string, string, error) {
	redisMu.Lock()
	defer redisMu.Unlock()
	if redisServer == nil {
		s, err := miniredis.Run()
		if err != nil {
			return "", "", fmt.Errorf("testenv: start miniredis: %w", err)
		}
		redisServer = s
	}
	return redisServer.Host(), redisServer.Port(), nil
}

func stopShared() {
	redisMu.Lock()
	defer redisMu.Unlock()
	if redisServer != nil {
		redisServer.Close()
		redisServer = nil
	}
}

// StartRedis is the TestMain variant of Redis. If REDIS_HOST is unset it
// starts the shared miniredis server and sets REDIS_HOST/REDIS_PORT; the
// returned cleanup stops the server and restores both variables. If REDIS_HOST
// is set it does nothing and cleanup is a no-op.
func StartRedis() (cleanup func(), err error) {
	if os.Getenv(redisHostEnv) != "" {
		return func() {}, nil
	}

	prevPort, hadPort := os.LookupEnv(redisPortEnv)
	host, port, err := startShared()
	if err != nil {
		return nil, err
	}
	if err := os.Setenv(redisHostEnv, host); err != nil {
		stopShared()
		return nil, err
	}
	if err := os.Setenv(redisPortEnv, port); err != nil {
		_ = os.Unsetenv(redisHostEnv)
		stopShared()
		return nil, err
	}

	return func() {
		stopShared()
		_ = os.Unsetenv(redisHostEnv)
		if hadPort {
			_ = os.Setenv(redisPortEnv, prevPort)
		} else {
			_ = os.Unsetenv(redisPortEnv)
		}
	}, nil
}

// Redis makes Redis available to t. If REDIS_HOST is unset it starts (or
// reuses) the shared miniredis server and sets REDIS_HOST/REDIS_PORT for the
// duration of t via t.Setenv, so t must not be parallel. The server itself
// stays up until the test binary exits, keeping pkg/db's client valid.
func Redis(t testing.TB) {
	t.Helper()
	if os.Getenv(redisHostEnv) != "" {
		return
	}
	host, port, err := startShared()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(redisHostEnv, host)
	t.Setenv(redisPortEnv, port)
}

// Postgres skips t unless DATABASE_URL is set.
func Postgres(t testing.TB) {
	t.Helper()
	Env(t, DatabaseURLEnv)
}

// Env skips t unless every named environment variable is set and non-empty.
func Env(t testing.TB, names ...string) {
	t.Helper()
	var missing []string
	for _, name := range names {
		if os.Getenv(name) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Skipf("testenv: skipping, required env not set: %s", strings.Join(missing, ", "))
	}
}
