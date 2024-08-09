package db

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRediscribe(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	done := make(chan struct{})
	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"test-channel"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			assert.Equal(t, "test-channel", msg.Channel)
			assert.Equal(t, "test-message", msg.Payload)
			close(done)
			return nil
		}),
	)
	require.NoError(t, err)

	go rediscriber.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Publish a test message
	err = rediscriber.client.Publish(ctx, "test-channel", "test-message").Err()
	require.NoError(t, err)

	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for message processing")
	}
}

func TestRediscribeErrorHandling(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	errorHandlerCalled := false
	errorHandled := make(chan struct{})
	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"error-channel"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			errorHandlerCalled = true
			close(errorHandled)
			return assert.AnError
		}),
	)
	require.NoError(t, err)

	// Start the Rediscriber
	go rediscriber.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Publish a test message
	err = rediscriber.client.Publish(ctx, "error-channel", "test-message").Err()
	require.NoError(t, err)

	// Wait for the error handler to be called or timeout
	select {
	case <-errorHandled:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for error handling")
	}

	assert.True(t, errorHandlerCalled, "The error handler should have been called")
}

func TestRediscribeReconnectOnConnectionFailure(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Use invalid Redis host/port to simulate connection failure
	host := "invalid-host"
	port := "1234"

	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"test-channel"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			return nil
		}),
		WithReconnectInterval(100*time.Millisecond),
	)
	require.NoError(t, err)

	// Start the Rediscriber and allow it to try reconnecting
	go rediscriber.Start(ctx)

	// Give it some time to attempt reconnecting
	time.Sleep(200 * time.Millisecond)

	// Assert that the client is still nil, as it couldn't connect
	assert.Nil(t, rediscriber.client, "Redis client should be nil after failed connection attempts")
}

func TestRediscribeGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"test-channel"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			return nil
		}),
	)
	require.NoError(t, err)

	go rediscriber.Start(ctx)

	// Simulate some work before shutting down
	time.Sleep(100 * time.Millisecond)

	// Cancel the context to trigger shutdown
	cancel()

	// Give the Rediscriber time to shut down
	time.Sleep(100 * time.Millisecond)

	// Assert that the client is nil after shutdown
	assert.Nil(t, rediscriber.client, "Redis client should be nil after shutdown")
}

func TestRediscribeResumeSubscriptionAfterReconnection(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	done := make(chan struct{})
	reconnected := make(chan struct{})
	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"test-channel"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			assert.Equal(t, "test-channel", msg.Channel)
			assert.Equal(t, "test-message", msg.Payload)
			close(done)
			return nil
		}),
		WithReconnectInterval(100*time.Millisecond),
	)
	require.NoError(t, err)

	go func() {
		rediscriber.Start(ctx)
		close(reconnected) // Signal that Start has returned (unexpected)
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate Redis going down
	rediscriber.mu.Lock()
	rediscriber.client.Close()
	rediscriber.client = nil
	rediscriber.mu.Unlock()

	// Wait for reconnect and subscription resumption
	time.Sleep(500 * time.Millisecond)

	// Publish a test message after reconnection
	err = rediscriber.client.Publish(ctx, "test-channel", "test-message").Err()
	require.NoError(t, err)

	select {
	case <-done:
		return
	case <-reconnected:
		t.Fatal("Rediscriber unexpectedly stopped")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for message processing")
	}
}

func TestRediscribeMultipleChannels(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	done := make(chan struct{})
	messagesReceived := make(map[string]string)
	var mu sync.Mutex

	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"channel-1", "channel-2"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			mu.Lock()
			defer mu.Unlock()
			messagesReceived[msg.Channel] = msg.Payload
			if len(messagesReceived) == 2 {
				close(done)
			}
			return nil
		}),
	)
	require.NoError(t, err)

	go rediscriber.Start(ctx)

	// Wait until subscriptions are likely active
	time.Sleep(100 * time.Millisecond)

	// Publish messages to both channels
	err = rediscriber.client.Publish(ctx, "channel-1", "message-1").Err()
	require.NoError(t, err)

	err = rediscriber.client.Publish(ctx, "channel-2", "message-2").Err()
	require.NoError(t, err)

	select {
	case <-done:
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, "message-1", messagesReceived["channel-1"])
		assert.Equal(t, "message-2", messagesReceived["channel-2"])
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for message processing")
	}
}

func TestRediscribeRouterErrorHandling(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	errorHandled := make(chan struct{})
	rediscriber, err := NewRediscribe(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisTopics([]string{"error-channel"}),
		WithRedisRouter(func(ctx context.Context, msg *redis.Message) error {
			close(errorHandled)
			return assert.AnError // Simulate router error
		}),
	)
	require.NoError(t, err)

	go rediscriber.Start(ctx)

	// Wait until subscription is likely active
	time.Sleep(100 * time.Millisecond)

	// Publish a test message that triggers the error
	err = rediscriber.client.Publish(ctx, "error-channel", "test-message").Err()
	require.NoError(t, err)

	select {
	case <-errorHandled:
		// Router error was handled
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for error handling")
	}
}
