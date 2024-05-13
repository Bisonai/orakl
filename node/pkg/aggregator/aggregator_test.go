//nolint:all
package aggregator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
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

	_, err = NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config)
	if err != nil {
		t.Fatal("error creating new node")
	}
}

func TestNewAggregator_Error(t *testing.T) {
	_, err := NewAggregator(nil, nil, "", Config{})
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config)
	if err != nil {
		t.Fatal("error creating new node")
	}

	err = node.LeaderJob()
	if err != nil {
		t.Fatal("error running leader job")
	}
	assert.Greater(t, node.RoundID, int32(0), "RoundID should be greater than 0")
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config)

	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test_pair"

	val, dbTime, err := GetLatestLocalAggregate(ctx, node.ID)
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config)
	if err != nil {
		t.Fatal("error creating new node")
	}

	node.Name = "test_pair"

	roundId, err := getLatestRoundId(ctx, node.ID)
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config)
	if err != nil {
		t.Fatal("error creating new node")
	}

	err = InsertGlobalAggregate(ctx, node.ID, 20, 2, time.Now())
	if err != nil {
		t.Fatal("error inserting global aggregate")
	}

	roundId, err := getLatestRoundId(ctx, node.ID)
	if err != nil {
		t.Fatal("error getting latest round id")
	}

	redisResult, err := getLatestGlobalAggregateFromRdb(ctx, node.ID)
	if err != nil {
		t.Fatal("error getting latest global aggregate from rdb")
	}
	assert.Equal(t, int64(20), redisResult.Value)
	assert.Equal(t, int32(2), redisResult.Round)
	assert.Equal(t, int32(2), roundId)
}

func TestInsertProof(t *testing.T) {
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config)
	if err != nil {
		t.Fatal("error creating new node")
	}

	value := int64(20)
	round := int32(2)
	p, err := node.SignHelper.MakeGlobalAggregateProof(value, time.Now(), "test-aggregate")
	if err != nil {
		t.Fatal("error making global aggregate proof")
	}

	err = InsertProof(ctx, node.ID, round, [][]byte{p, p})
	if err != nil {
		t.Fatal("error inserting proof")
	}

	rdbResult, err := getProofFromRdb(ctx, node.ID, round)
	if err != nil {
		t.Fatal("error getting proof from rdb")
	}

	assert.EqualValues(t, bytes.Join([][]byte{p, p}, nil), rdbResult.Proof)

	pgsqlResult, err := getProofFromPgsql(ctx, node.ID, round)
	if err != nil {
		t.Fatal("error getting proof from pgsql:" + err.Error())
	}

	assert.EqualValues(t, bytes.Join([][]byte{p, p}, nil), pgsqlResult.Proof)

	err = db.QueryWithoutResult(ctx, "DELETE FROM proofs", nil)
	if err != nil {
		t.Fatal("error deleting proofs")
	}
}
