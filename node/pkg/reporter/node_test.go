//nolint:all
package reporter

import (
	"context"
	"math/big"
	"testing"

	"bisonai.com/orakl/node/pkg/raft"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	_, err = NewNode(ctx, testItems.reporterNode.Raft.Host, testItems.reporterNode.Raft.Ps)
	if err != nil {
		t.Fatal("error creating new reporter")
	}
}

func TestLeaderJob(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.reporterNode.leaderJob()
	if err != nil {
		t.Fatal("error running leader job")
	}
}

func TestResignLeader(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	testItems.reporterNode.resignLeader()
	assert.Equal(t, testItems.reporterNode.Raft.GetRole(), raft.Follower)
}

func TestHandleCustomMessage(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.reporterNode.handleCustomMessage(raft.Message{})
	assert.Equal(t, err.Error(), "unknown message type")
}

func TestGetLatestGlobalAggregates(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	result, err := testItems.reporterNode.getLatestGlobalAggregates(ctx)
	if err != nil {
		t.Fatal("error getting latest global aggregates")
	}
	assert.Equal(t, result[0], testItems.tmpData.globalAggregate)
}

func TestFilterInvalidAggregates(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	aggregates := []GlobalAggregate{{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}}
	result := testItems.reporterNode.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, aggregates)

	testItems.reporterNode.lastSubmissions = map[string]int64{"test-aggregate": 1}
	result = testItems.reporterNode.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, []GlobalAggregate{})
}

func TestIsAggValid(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	agg := GlobalAggregate{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}
	result := testItems.reporterNode.isAggValid(agg)
	assert.Equal(t, result, true)

	testItems.reporterNode.lastSubmissions = map[string]int64{"test-aggregate": 1}
	result = testItems.reporterNode.isAggValid(agg)
	assert.Equal(t, result, false)
}

func TestMakeContractArgs(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	agg := GlobalAggregate{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}
	pairs, values, err := testItems.reporterNode.makeContractArgs([]GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error making contract args")
	}

	assert.Equal(t, pairs[0], "test-aggregate")
	assert.Equal(t, values[0], big.NewInt(15))
}
