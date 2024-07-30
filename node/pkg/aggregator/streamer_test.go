//nolint:all
package aggregator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStreamer(t *testing.T) {
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

	_ = NewStreamer(WithConfigIds([]int32{testItems.tmpData.config.ID}))
	if err != nil {
		t.Fatal("error creating new node")
	}
}

func TestStreamerStart(t *testing.T) {
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

	streamer := NewStreamer(WithConfigIds([]int32{testItems.tmpData.config.ID}))

	streamer.Start(ctx)

	assert.NotEqual(t, nil, streamer.ctx)
}

func TestStreamerStop(t *testing.T) {
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

	streamer := NewStreamer(WithConfigIds([]int32{testItems.tmpData.config.ID}))

	streamer.Start(ctx)

	streamer.Stop()

	assert.Equal(t, nil, streamer.ctx)
}

func TestStreamerDataStore(t *testing.T) {
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

	streamer := NewStreamer(WithConfigIds([]int32{testItems.tmpData.globalAggregate.ConfigID}))

	streamer.Start(ctx)
	defer streamer.Stop()
	assert.NotEqual(t, nil, streamer.ctx)

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
		ConfigID: testItems.tmpData.globalAggregate.ConfigID,
		Round:    testItems.tmpData.globalAggregate.Round,
		Proof:    concatProof,
	}

	time.Sleep(time.Millisecond * 50)
	err = PublishGlobalAggregateAndProof(ctx, testItems.tmpData.globalAggregate, proof)
	if err != nil {
		t.Fatal("error publishing global aggregate and proof")
	}
	time.Sleep(time.Second * 3)

	updateTime, value, exists := streamer.LatestGlobalAggregateInfo.GetData(testItems.tmpData.globalAggregate.ConfigID)
	assert.True(t, exists)

	assert.NotEqual(t, nil, updateTime)
	assert.Equal(t, testItems.tmpData.globalAggregate.Value, *value)

	rdbLoadedProof, err := getProofFromRdb(ctx, testItems.tmpData.globalAggregate.ConfigID, testItems.tmpData.globalAggregate.Round)
	if err != nil {
		t.Fatal("error getting proof from rdb")
	}
	assert.Equal(t, proof, rdbLoadedProof)

	rdbLoadedGlobalAggregate, err := getLatestGlobalAggregateFromRdb(ctx, testItems.tmpData.globalAggregate.ConfigID)
	if err != nil {
		t.Fatal("error getting global aggregate from rdb")
	}
	assert.Equal(t, testItems.tmpData.globalAggregate.Value, rdbLoadedGlobalAggregate.Value)
	assert.Equal(t, testItems.tmpData.globalAggregate.Timestamp.UTC(), rdbLoadedGlobalAggregate.Timestamp.UTC())
	assert.Equal(t, testItems.tmpData.globalAggregate.Round, rdbLoadedGlobalAggregate.Round)

	pgsLoadedProof, err := getProofFromPgs(ctx, testItems.tmpData.globalAggregate.ConfigID, testItems.tmpData.globalAggregate.Round)
	if err != nil {
		t.Fatal("error getting proof from pgs:" + err.Error())
	}
	assert.Equal(t, proof, pgsLoadedProof)

	pgsLoadedGlobalAggregate, err := getLatestGlobalAggregateFromPgs(ctx, testItems.tmpData.globalAggregate.ConfigID)
	if err != nil {
		t.Fatal("error getting global aggregate from pgs" + err.Error())
	}
	assert.Equal(t, testItems.tmpData.globalAggregate, pgsLoadedGlobalAggregate)
}
