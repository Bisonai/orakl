//nolint:all
package aggregator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAggregator(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	_, err = NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}
}

func TestNewAggregator_Error(t *testing.T) {
	_, err := NewAggregator(nil, nil, "")
	assert.NotNil(t, err, "Expected error when creating new aggregator with nil parameters")
}

func TestLeaderJob(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	err = node.LeaderJob()
	if err != nil {
		t.Fatal("error running leader job")
	}
	assert.Greater(t, node.RoundID, int64(0), "RoundID should be greater than 0")
}

func TestGetLatestLocalAggregate(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)

	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test_pair"

	val, dbTime, err := GetLatestLocalAggregate(ctx, node.Name)
	if err != nil {
		t.Fatal("error getting latest local aggregate")
	}

	assert.NotZero(t, dbTime, "dbTime should not be zero")

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
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test_pair"

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
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test_pair"

	err = InsertGlobalAggregate(ctx, node.Name, 20, 2)
	if err != nil {
		t.Fatal("error inserting global aggregate")
	}

	roundId, err := node.getLatestRoundId(ctx)
	if err != nil {
		t.Fatal("error getting latest round id")
	}

	redisResult, err := GetLatestGlobalAggregateFromRdb(ctx, "test_pair")
	if err != nil {
		t.Fatal("error getting latest global aggregate from rdb")
	}
	assert.Equal(t, int64(20), redisResult.Value)
	assert.Equal(t, int64(2), redisResult.Round)

	assert.Equal(t, int64(2), roundId)
}
