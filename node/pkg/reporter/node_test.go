//nolint:all
package reporter

import (
	"context"
	"math/big"
	"testing"

	"bisonai.com/orakl/node/pkg/raft"
	"github.com/klaytn/klaytn/common"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	_, err = NewNode(ctx, testItems.app.Host, testItems.app.Pubsub)
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
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	err = testItems.app.Reporter.leaderJob()
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
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	testItems.app.Reporter.resignLeader()
	assert.Equal(t, testItems.app.Reporter.Raft.GetRole(), raft.Follower)
}

func TestHandleCustomMessage(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	err = testItems.app.Reporter.handleCustomMessage(raft.Message{})
	assert.Equal(t, err.Error(), "unknown message type")
}

func TestGetLatestGlobalAggregates(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	result, err := testItems.app.Reporter.getLatestGlobalAggregates(ctx)
	if err != nil {
		t.Fatal("error getting latest global aggregates")
	}
	assert.Equal(t, result[0].Name, testItems.tmpData.globalAggregate.Name)
	assert.Equal(t, result[0].Value, testItems.tmpData.globalAggregate.Value)
}

func TestFilterInvalidAggregates(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	aggregates := []GlobalAggregate{{
		Name:    "test-aggregate",
		Value:   15,
		Round:   1,
		Address: "0x1234",
	}}
	result := testItems.app.Reporter.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, aggregates)

	testItems.app.Reporter.lastSubmissions = map[string]int64{"test-aggregate": 1}
	result = testItems.app.Reporter.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, []GlobalAggregate{})
}

func TestIsAggValid(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	agg := GlobalAggregate{
		Name:    "test-aggregate",
		Value:   15,
		Round:   1,
		Address: "0x1234",
	}
	result := testItems.app.Reporter.isAggValid(agg)
	assert.Equal(t, result, true)

	testItems.app.Reporter.lastSubmissions = map[string]int64{"test-aggregate": 1}
	result = testItems.app.Reporter.isAggValid(agg)
	assert.Equal(t, result, false)
}

func TestMakeContractArgs(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	agg := GlobalAggregate{
		Name:    "test-aggregate",
		Value:   15,
		Round:   1,
		Address: "0x1234",
	}
	addresses, values, err := testItems.app.Reporter.makeContractArgs([]GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error making contract args")
	}

	assert.Equal(t, addresses[0], common.HexToAddress(agg.Address))
	assert.Equal(t, values[0], big.NewInt(15))
}
