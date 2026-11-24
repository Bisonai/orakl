package testenv

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisReachable(t *testing.T) {
	Redis(t)
	host, port := os.Getenv(redisHostEnv), os.Getenv(redisPortEnv)
	if host == "" || port == "" {
		t.Fatalf("REDIS_HOST/REDIS_PORT not set: %q %q", host, port)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{Addr: host + ":" + port})
	defer client.Close()

	if err := client.Set(ctx, "testenv:key", "v", time.Minute).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got, err := client.Get(ctx, "testenv:key").Result(); err != nil || got != "v" {
		t.Fatalf("get: %q %v", got, err)
	}

	sub := client.Subscribe(ctx, "testenv:ch")
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := client.Publish(ctx, "testenv:ch", "hello").Err(); err != nil {
		t.Fatalf("publish: %v", err)
	}
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil || msg.Payload != "hello" {
		t.Fatalf("receive: %v %v", msg, err)
	}
}

func TestRedisKeepsExistingHost(t *testing.T) {
	t.Setenv(redisHostEnv, "redis.example")
	t.Setenv(redisPortEnv, "6390")
	Redis(t)
	if got := os.Getenv(redisHostEnv); got != "redis.example" {
		t.Fatalf("REDIS_HOST overwritten: %q", got)
	}

	cleanup, err := StartRedis()
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	if got := os.Getenv(redisPortEnv); got != "6390" {
		t.Fatalf("REDIS_PORT changed: %q", got)
	}
}

func TestStartRedisCleanupRestoresEnv(t *testing.T) {
	stopShared() // start from no server so StartRedis owns the one it starts
	t.Cleanup(stopShared)
	t.Setenv(redisHostEnv, "")
	t.Setenv(redisPortEnv, "")
	_ = os.Unsetenv(redisPortEnv)

	cleanup, err := StartRedis()
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv(redisHostEnv) == "" || os.Getenv(redisPortEnv) == "" {
		t.Fatal("StartRedis did not set REDIS_HOST/REDIS_PORT")
	}
	cleanup()
	if host, ok := os.LookupEnv(redisHostEnv); !ok || host != "" {
		t.Fatalf("REDIS_HOST not restored to empty: %q %v", host, ok)
	}
	if _, ok := os.LookupEnv(redisPortEnv); ok {
		t.Fatal("REDIS_PORT not unset")
	}
	if _, _, created, err := startShared(); err != nil || !created {
		t.Fatalf("StartRedis cleanup did not stop the server it started: created=%v err=%v", created, err)
	}
}

func TestStartRedisKeepsServerStartedByRedis(t *testing.T) {
	stopShared() // start from no server so Redis(t) owns the one it starts
	t.Cleanup(stopShared)
	t.Setenv(redisHostEnv, "")
	t.Run("redis", func(t *testing.T) { Redis(t) })

	host, port, _, err := startShared()
	if err != nil {
		t.Fatal(err)
	}
	cleanup, err := StartRedis()
	if err != nil {
		t.Fatal(err)
	}
	cleanup()

	h, p, created, err := startShared()
	if err != nil {
		t.Fatal(err)
	}
	if created || h != host || p != port {
		t.Fatalf("shared server was stopped by StartRedis cleanup: created=%v %s:%s vs %s:%s", created, h, p, host, port)
	}
}

func TestEnvSkipsWhenMissing(t *testing.T) {
	t.Setenv(RequireEnv, "") // CI may set it for the whole run
	t.Setenv("TESTENV_PRESENT", "1")
	t.Setenv("TESTENV_MISSING", "")

	skipped := t.Run("missing", func(t *testing.T) {
		Env(t, "TESTENV_PRESENT", "TESTENV_MISSING")
		t.Fatal("Env did not skip")
	})
	if !skipped {
		t.Fatal("subtest failed instead of skipping")
	}

	ran := false
	t.Run("present", func(t *testing.T) {
		Env(t, "TESTENV_PRESENT")
		ran = true
	})
	if !ran {
		t.Fatal("Env skipped although every variable was set")
	}
}

func TestStartRedisRejectsHostWithoutPort(t *testing.T) {
	t.Setenv(redisHostEnv, "redis.example")
	t.Setenv(redisPortEnv, "")
	if _, err := StartRedis(); err == nil {
		t.Fatal("expected error for REDIS_HOST without REDIS_PORT")
	}
}

// recordTB observes Skip vs Fatal without failing the real test. Like
// testing.T, Skipf and Fatalf stop the calling goroutine.
type recordTB struct {
	testing.TB
	skipped, failed bool
	msg             string
}

func (r *recordTB) Helper() {}

func (r *recordTB) Skipf(format string, args ...any) {
	r.skipped, r.msg = true, fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func (r *recordTB) Fatalf(format string, args ...any) {
	r.failed, r.msg = true, fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// observe runs fn against a recordTB on its own goroutine (Goexit-safe).
func observe(t *testing.T, fn func(testing.TB)) *recordTB {
	r := &recordTB{TB: t}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(r)
	}()
	<-done
	return r
}

func TestRequireTestEnv(t *testing.T) {
	t.Setenv("TESTENV_PRESENT", "1")
	t.Setenv("TESTENV_MISSING", "")
	t.Setenv(DatabaseURLEnv, "")

	helpers := map[string]func(testing.TB){
		"Env":      func(tb testing.TB) { Env(tb, "TESTENV_PRESENT", "TESTENV_MISSING") },
		"Postgres": func(tb testing.TB) { Postgres(tb) },
	}
	wantVar := map[string]string{"Env": "TESTENV_MISSING", "Postgres": DatabaseURLEnv}

	for _, tc := range []struct {
		value    string
		wantFail bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"FALSE", false},
		{"1", true},
		{"true", true},
		{"yes", true},
	} {
		for name, fn := range helpers {
			t.Run(fmt.Sprintf("%s/REQUIRE_TEST_ENV=%q", name, tc.value), func(t *testing.T) {
				t.Setenv(RequireEnv, tc.value)
				r := observe(t, fn)
				if r.failed != tc.wantFail || r.skipped == tc.wantFail {
					t.Fatalf("failed=%v skipped=%v, want fail=%v", r.failed, r.skipped, tc.wantFail)
				}
				if !strings.Contains(r.msg, wantVar[name]) {
					t.Fatalf("message %q does not name %s", r.msg, wantVar[name])
				}
				if tc.wantFail && !strings.Contains(r.msg, RequireEnv) {
					t.Fatalf("failure message %q does not mention %s", r.msg, RequireEnv)
				}
			})
		}
	}

	t.Run("present vars pass under REQUIRE_TEST_ENV", func(t *testing.T) {
		t.Setenv(RequireEnv, "1")
		r := observe(t, func(tb testing.TB) { Env(tb, "TESTENV_PRESENT") })
		if r.failed || r.skipped {
			t.Fatalf("failed=%v skipped=%v with every variable set: %s", r.failed, r.skipped, r.msg)
		}
	})
}
