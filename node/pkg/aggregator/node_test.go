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

	_, err = NewNode(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}
}

func TestLeaderJob(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
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

	node, err := NewNode(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)

	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test-aggregate"

	val, dbTime, err := node.getLatestLocalAggregate(ctx)
	if err != nil {
		t.Fatal("error getting latest local aggregate")
	}

	assert.Equal(t, val, testItems.tmpData.rLocalAggregate.Value)
	assert.Equal(t, val, testItems.tmpData.pLocalAggregate.Value)

	assert.Equal(t, dbTime.UTC().Truncate(time.Millisecond), testItems.tmpData.rLocalAggregate.Timestamp.UTC().Truncate(time.Millisecond))
	assert.Equal(t, dbTime.UTC().Truncate(time.Millisecond), testItems.tmpData.pLocalAggregate.Timestamp.UTC().Truncate(time.Millisecond))
}

func TestGetLatestRoundId(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	node, err := NewNode(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test-aggregate"

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

	node, err := NewNode(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test-aggregate"

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
