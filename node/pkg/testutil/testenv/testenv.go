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
// themselves (db.CloseRedis) before calling the StartRedis cleanup. pkg/db
// caches its client, so a test that later points REDIS_HOST elsewhere must call
// db.CloseRedis first or it keeps talking to the previous server.
//
// miniredis limitation: keys with a TTL do not expire on wall-clock time; they
// only expire when the server clock is advanced. Tests that depend on expiry
// should not rely on the miniredis fallback (gate them with Env on REDIS_HOST).
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
// returns its host and port; created reports whether this call started it.
func startShared() (host, port string, created bool, err error) {
	redisMu.Lock()
	defer redisMu.Unlock()
	if redisServer == nil {
		s, err := miniredis.Run()
		if err != nil {
			return "", "", false, fmt.Errorf("testenv: start miniredis: %w", err)
		}
		redisServer = s
		created = true
	}
	return redisServer.Host(), redisServer.Port(), created, nil
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
// returned cleanup restores both variables and stops the server if this call
// started it (a server started earlier by Redis(t) stays up). If REDIS_HOST is
// set it does nothing and cleanup is a no-op.
func StartRedis() (cleanup func(), err error) {
	if os.Getenv(redisHostEnv) != "" {
		if err := checkRedisPort(); err != nil {
			return nil, err
		}
		return func() {}, nil
	}

	prevHost, hadHost := os.LookupEnv(redisHostEnv)
	prevPort, hadPort := os.LookupEnv(redisPortEnv)
	host, port, created, err := startShared()
	if err != nil {
		return nil, err
	}
	cleanup = func() {
		if created {
			stopShared()
		}
		restoreEnv(redisHostEnv, prevHost, hadHost)
		restoreEnv(redisPortEnv, prevPort, hadPort)
	}
	if err := os.Setenv(redisHostEnv, host); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.Setenv(redisPortEnv, port); err != nil {
		cleanup()
		return nil, err
	}
	return cleanup, nil
}

// checkRedisPort reports a half-configured external Redis (REDIS_HOST set,
// REDIS_PORT missing) instead of letting pkg/db fail later.
func checkRedisPort() error {
	if os.Getenv(redisPortEnv) == "" {
		return fmt.Errorf("testenv: %s is set but %s is not; set both or unset %s to use miniredis", redisHostEnv, redisPortEnv, redisHostEnv)
	}
	return nil
}

func restoreEnv(name, value string, had bool) {
	if had {
		_ = os.Setenv(name, value)
	} else {
		_ = os.Unsetenv(name)
	}
}

// Redis makes Redis available to t. If REDIS_HOST is unset it starts (or
// reuses) the shared miniredis server and sets REDIS_HOST/REDIS_PORT for the
// duration of t via t.Setenv, so t must not be parallel. The server itself
// stays up until the test binary exits, keeping pkg/db's client valid.
func Redis(t testing.TB) {
	t.Helper()
	if os.Getenv(redisHostEnv) != "" {
		if err := checkRedisPort(); err != nil {
			t.Fatal(err)
		}
		return
	}
	host, port, _, err := startShared()
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
