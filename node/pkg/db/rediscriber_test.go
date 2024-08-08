package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRediscriber(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	done := make(chan struct{})
	rediscriber, err := NewRediscriber(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisChannels([]string{"test-channel"}),
		WithRedisRouter(func(msg *redis.Message) error {
			assert.Equal(t, "test-channel", msg.Channel)
			assert.Equal(t, "test-message", msg.Payload)
			close(done)
			return nil
		}),
	)
	require.NoError(t, err)
	defer rediscriber.client.Close()

	// Start the Rediscriber
	started := make(chan struct{})
	go func() {
		err := rediscriber.Start(ctx)
		require.NoError(t, err)
		close(started)
	}()

	<-started

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

func TestRediscriber_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	errorHandlerCalled := false
	errorHandled := make(chan struct{})
	rediscriber, err := NewRediscriber(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisChannels([]string{"error-channel"}),
		WithRedisRouter(func(msg *redis.Message) error {
			errorHandlerCalled = true
			close(errorHandled)
			return assert.AnError
		}),
	)
	require.NoError(t, err)
	defer rediscriber.client.Close()

	// Start the Rediscriber
	started := make(chan struct{})
	go func() {
		err := rediscriber.Start(ctx)
		require.NoError(t, err)
		close(started)
	}()
	<-started

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
