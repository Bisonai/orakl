//nolint:all
package test

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal/hub"
	"bisonai.com/miko/node/pkg/wss"
	"github.com/stretchr/testify/assert"
)

func TestCollectorStartAndStop(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	time.Sleep(10 * time.Millisecond)

	collector := testItems.Collector
	assert.True(t, collector.IsRunning)

	assert.Greater(t, len(collector.FeedHashes), 0)
	collector.Stop()
	assert.False(t, collector.IsRunning)
}

func TestCollectorStream(t *testing.T) {
	ctx := context.Background()
	clean, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := clean(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	time.Sleep(20 * time.Millisecond)

	collector := testItems.Collector
	assert.Greater(t, len(collector.FeedHashes), 0)
	assert.True(t, collector.IsRunning)

	headers := map[string]string{"X-API-Key": testItems.ApiKey}
	conn, err := wss.NewWebsocketHelper(ctx, wss.WithEndpoint(testItems.MockDal.URL+"/ws"), wss.WithRequestHeaders(headers))
	if err != nil {
		t.Fatalf("error creating websocket helper: %v", err)
	}

	err = conn.Dial(ctx)
	if err != nil {
		t.Fatalf("error dialing websocket: %v", err)
	}

	err = conn.Write(ctx, hub.Subscription{
		Method: "SUBSCRIBE",
		Params: []string{"submission@test-aggregate"},
	})
	if err != nil {
		t.Fatalf("error subscribing to websocket: %v", err)
	}

	ch := make(chan any, 16)
	go conn.Read(ctx, ch)

	result := awaitWSSubmission(ctx, t, testItems, ch, "test-aggregate")
	assert.Equal(t, "test-aggregate", result.Symbol)
	assert.Equal(t, "15", result.Value)

	err = conn.Close()
	if err != nil {
		t.Fatalf("error closing websocket: %v", err)
	}
}
