//nolint:all
package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewNode(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	_, err = NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}
}

func TestGetLeaderJobTimeout(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	assert.Equal(t, node.GetLeaderJobTimeout(), node.LeaderJobTimeout)
}

func TestGetLeaderJobTicker(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	assert.Equal(t, node.GetLeaderJobTicker(), node.LeaderJobTicker)
}

func TestSetLeaderJobTicker(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	duration := 10 * time.Second
	err = node.SetLeaderJobTicker(&duration)
	if err != nil {
		t.Fatal("error setting leader job ticker")
	}

	assert.Equal(t, node.LeaderJobTicker, time.NewTicker(duration))
}

func TestLeaderJob(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	err = node.LeaderJob()
	if err != nil {
		t.Fatal("error running leader job")
	}
}

func TestGetLatestLocalAggregate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test"

	val, time, err := node.getLatestLocalAggregate(ctx)
	if err != nil {
		t.Fatal("error getting latest local aggregate")
	}

	assert.Equal(t, val, testItems.tmpData.rLocalAggregate.Value)
	assert.Equal(t, val, testItems.tmpData.pLocalAggregate.Value)
	assert.Equal(t, time, testItems.tmpData.rLocalAggregate.Timestamp)
	assert.Equal(t, time, testItems.tmpData.pLocalAggregate.Timestamp)
}

func TestGetLatestRoundId(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test"

	roundId, err := node.getLatestRoundId(ctx)
	if err != nil {
		t.Fatal("error getting latest round id")
	}

	assert.Equal(t, roundId, testItems.tmpData.globalAggregate.Round)
}

func TestInsertGlobalAggregate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(*testItems.host, testItems.pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test"

	err = node.insertGlobalAggregate(20, 2)
	if err != nil {
		t.Fatal("error inserting global aggregate")
	}

	roundId, err := node.getLatestRoundId(ctx)
	if err != nil {
		t.Fatal("error getting latest round id")
	}

	assert.Equal(t, roundId, int64(2))
}
