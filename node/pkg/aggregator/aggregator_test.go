//nolint:all
package aggregator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/common/keys"
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

	_, err = NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config, testItems.signer, testItems.latestLocalAggMap)
	if err != nil {
		t.Fatal("error creating new node")
	}
}

func TestNewAggregator_Error(t *testing.T) {
	_, err := NewAggregator(nil, nil, "", Config{}, nil, nil)
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config, testItems.signer, testItems.latestLocalAggMap)
	if err != nil {
		t.Fatal("error creating new node")
	}

	err = node.LeaderJob(ctx)
	if err != nil {
		t.Fatal("error running leader job")
	}
	assert.Greater(t, node.RoundID, int32(0), "RoundID should be greater than 0")
}

func TestGetLatestLocalAggregate(t *testing.T) {
	t.Skip()
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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config, testItems.signer, testItems.latestLocalAggMap)

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

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config, testItems.signer, testItems.latestLocalAggMap)
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

func TestPublishGlobalAggregateAndProof(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		cancel()
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config, testItems.signer, testItems.latestLocalAggMap)
	if err != nil {
		t.Fatal("error creating new node")
	}

	p, err := node.Signer.MakeGlobalAggregateProof(
		testItems.tmpData.globalAggregate.Value,
		testItems.tmpData.globalAggregate.Timestamp,
		"test_pair",
	)
	if err != nil {
		t.Fatal("error making global aggregate proof")
	}

	concatProof := bytes.Join([][]byte{p, p}, nil)

	proof := Proof{
		ConfigID: node.ID,
		Round:    testItems.tmpData.globalAggregate.Round,
		Proof:    concatProof,
	}

	ch := make(chan SubmissionData)
	err = db.Subscribe(ctx, keys.SubmissionDataStreamKey(node.ID), ch)
	if err != nil {
		t.Fatal("error subscribing to stream")
	}

	err = PublishGlobalAggregateAndProof(ctx, testItems.tmpData.globalAggregate, proof)
	if err != nil {
		t.Fatal("error publishing global aggregate and proof")
	}

	data := <-ch
	assert.EqualValues(t, proof, data.Proof)
	assert.Equal(t, testItems.tmpData.globalAggregate.Round, data.GlobalAggregate.Round)
	assert.Equal(t, testItems.tmpData.globalAggregate.Value, data.GlobalAggregate.Value)
	assert.Equal(t, testItems.tmpData.globalAggregate.Timestamp.UTC(), data.GlobalAggregate.Timestamp.UTC())

}
