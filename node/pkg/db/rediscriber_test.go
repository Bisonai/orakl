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

	rediscriber, err := NewRediscriber(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisChannels([]string{"test-channel"}),
		WithRedisRouter(func(msg *redis.Message) error {
			assert.Equal(t, "test-channel", msg.Channel)
			assert.Equal(t, "test-message", msg.Payload)
			return nil
		}),
	)
	require.NoError(t, err)
	defer rediscriber.client.Close()

	// Start the Rediscriber
	go func() {
		err := rediscriber.Start(ctx)
		require.NoError(t, err)
	}()

	// Allow some time for the subscription to start
	time.Sleep(1 * time.Second)

	// Publish a test message
	err = rediscriber.client.Publish(ctx, "test-channel", "test-message").Err()
	require.NoError(t, err)

	// Allow some time for the message to be processed
	time.Sleep(1 * time.Second)
}

func TestRediscriber_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	errorHandlerCalled := false

	rediscriber, err := NewRediscriber(ctx,
		WithRedisHost(host),
		WithRedisPort(port),
		WithRedisChannels([]string{"error-channel"}),
		WithRedisRouter(func(msg *redis.Message) error {
			errorHandlerCalled = true
			return assert.AnError
		}),
	)
	require.NoError(t, err)
	defer rediscriber.client.Close()

	// Start the Rediscriber
	go func() {
		err := rediscriber.Start(ctx)
		require.NoError(t, err)
	}()

	// Allow some time for the subscription to start
	time.Sleep(1 * time.Second)

	// Publish a test message
	err = rediscriber.client.Publish(ctx, "error-channel", "test-message").Err()
	require.NoError(t, err)

	// Allow some time for the message to be processed
	time.Sleep(1 * time.Second)

	assert.True(t, errorHandlerCalled, "The error handler should have been called")
}
