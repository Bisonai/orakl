//nolint:all
package reporter

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"bisonai.com/orakl/node/pkg/raft"
	"github.com/klaytn/klaytn/common"
	"github.com/stretchr/testify/assert"
)

func TestNewReporter(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	_, err = NewReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
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
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	testItems.app.Reporter.SetKlaytnHelper(ctx)
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
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
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
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
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
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
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
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	aggregates := []GlobalAggregate{{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}}
	result := testItems.app.Reporter.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, aggregates)

	testItems.app.Reporter.SubmissionPairs = map[string]SubmissionPair{"test-aggregate": {LastSubmission: 1, Address: common.HexToAddress("0x1234")}}
	result = testItems.app.Reporter.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, []GlobalAggregate{})
}

func TestIsAggValid(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	agg := GlobalAggregate{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}
	result := testItems.app.Reporter.isAggValid(agg)
	assert.Equal(t, result, true)

	testItems.app.Reporter.SubmissionPairs = map[string]SubmissionPair{"test-aggregate": {LastSubmission: 1, Address: common.HexToAddress("0x1234")}}
	result = testItems.app.Reporter.isAggValid(agg)
	assert.Equal(t, result, false)
}

func TestMakeContractArgs(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()
	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)
	agg := GlobalAggregate{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}
	addresses, values, err := testItems.app.Reporter.makeContractArgs([]GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error making contract args")
	}

	assert.Equal(t, addresses[0], testItems.app.Reporter.SubmissionPairs[agg.Name].Address)
	assert.Equal(t, values[0], big.NewInt(15))
}

func TestGetLatestGlobalAggregatesRdb(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)

	result, err := testItems.app.Reporter.getLatestGlobalAggregatesRdb(ctx)
	if err != nil {
		t.Fatal("error getting latest global aggregates from rdb")
	}

	assert.Equal(t, result[0].Name, testItems.tmpData.globalAggregate.Name)
	assert.Equal(t, result[0].Value, testItems.tmpData.globalAggregate.Value)
}

func TestGetLatestGlobalAggregatesPgsql(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	testItems.app.setReporter(ctx, testItems.app.Host, testItems.app.Pubsub)

	result, err := testItems.app.Reporter.getLatestGlobalAggregatesPgsql(ctx)
	if err != nil {
		fmt.Println(err)
		t.Fatal("error getting latest global aggregates from pgs")
	}

	assert.Equal(t, result[0].Name, testItems.tmpData.globalAggregate.Name)
	assert.Equal(t, result[0].Value, testItems.tmpData.globalAggregate.Value)
}
