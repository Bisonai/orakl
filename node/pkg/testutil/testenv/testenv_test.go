package testenv

import (
	"context"
	"os"
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
	if os.Getenv(redisHostEnv) != "" {
		t.Fatal("REDIS_HOST not cleared")
	}
	if _, ok := os.LookupEnv(redisPortEnv); ok {
		t.Fatal("REDIS_PORT not unset")
	}
}

func TestEnvSkipsWhenMissing(t *testing.T) {
	t.Setenv("TESTENV_PRESENT", "1")
	t.Setenv("TESTENV_MISSING", "")

	skipped := t.Run("missing", func(t *testing.T) {
		Env(t, "TESTENV_PRESENT", "TESTENV_MISSING")
		t.Fatal("Env did not skip")
	})
	if !skipped {
		t.Fatal("subtest failed instead of skipping")
	}

	t.Run("present", func(t *testing.T) {
		Env(t, "TESTENV_PRESENT")
	})
}
