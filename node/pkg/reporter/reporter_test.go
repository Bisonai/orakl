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
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	submissionPairs, err := getSubmissionPairs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}
	groupedSubmissionPairs := groupSubmissionPairsByIntervals(submissionPairs)

	for groupInterval, pairs := range groupedSubmissionPairs {
		_, err := NewReporter(ctx, testItems.app.Host, testItems.app.Pubsub, pairs, groupInterval)
		if err != nil {
			t.Fatalf("error creating new reporter: %v", err)
		}
	}
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
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	reporter.SetKlaytnHelper(ctx)
	err = reporter.leaderJob()
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
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}
	reporter.resignLeader()
	assert.Equal(t, reporter.Raft.GetRole(), raft.Follower)
}

func TestHandleCustomMessage(t *testing.T) {
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
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	err = reporter.handleCustomMessage(raft.Message{})
	assert.Equal(t, err.Error(), "unknown message type")
}

func TestGetLatestGlobalAggregates(t *testing.T) {
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
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}
	result, err := reporter.getLatestGlobalAggregates(ctx)
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
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	aggregates := []GlobalAggregate{{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}}
	result := reporter.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, aggregates)

	reporter.SubmissionPairs = map[string]SubmissionPair{"test-aggregate": {LastSubmission: 1, Address: common.HexToAddress("0x1234")}}
	result = reporter.filterInvalidAggregates(aggregates)
	assert.Equal(t, result, []GlobalAggregate{})
}

func TestIsAggValid(t *testing.T) {
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
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	agg := GlobalAggregate{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}
	result := reporter.isAggValid(agg)
	assert.Equal(t, result, true)

	reporter.SubmissionPairs = map[string]SubmissionPair{"test-aggregate": {LastSubmission: 1, Address: common.HexToAddress("0x1234")}}
	result = reporter.isAggValid(agg)
	assert.Equal(t, result, false)
}

func TestMakeContractArgs(t *testing.T) {
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
	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	agg := GlobalAggregate{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}

	addresses, values, err := reporter.makeContractArgsWithoutProofs([]GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error making contract args")
	}

	assert.Equal(t, addresses[0], reporter.SubmissionPairs[agg.Name].Address)
	assert.Equal(t, values[0], big.NewInt(15))

	rawProofs, err := reporter.getProofsRdb(ctx, []GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error getting proofs")
	}

	proofMap := ProofsToMap(rawProofs)

	addresses, values, proofs, err := reporter.makeContractArgsWithProofs([]GlobalAggregate{agg}, proofMap)
	if err != nil {
		t.Fatal("error making contract args")
	}
	assert.Equal(t, reporter.SubmissionPairs[agg.Name].Address, addresses[0])
	assert.Equal(t, big.NewInt(15), values[0])

	proofArr := make([][]byte, len(proofs))
	for i, p := range rawProofs {
		proofArr[i] = p.Proof
	}

	assert.EqualValues(t, proofs, proofArr)
}

func TestGetLatestGlobalAggregatesRdb(t *testing.T) {
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

	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	result, err := reporter.getLatestGlobalAggregatesRdb(ctx)
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
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	result, err := reporter.getLatestGlobalAggregatesPgsql(ctx)
	if err != nil {
		fmt.Println(err)
		t.Fatal("error getting latest global aggregates from pgs")
	}

	assert.Equal(t, result[0].Name, testItems.tmpData.globalAggregate.Name)
	assert.Equal(t, result[0].Value, testItems.tmpData.globalAggregate.Value)
}

func TestGetProofsRdb(t *testing.T) {
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

	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	agg := testItems.tmpData.globalAggregate
	result, err := reporter.getProofsRdb(ctx, []GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error getting proofs from rdb")
	}
	assert.EqualValues(t, testItems.tmpData.proofBytes, result[0].Proof)
}

func TestGetProofsPgsql(t *testing.T) {
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

	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	agg := testItems.tmpData.globalAggregate
	result, err := reporter.getProofsPgsql(ctx, []GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error getting proofs from pgsql")
	}
	assert.EqualValues(t, testItems.tmpData.proofBytes, result[0].Proof)
}

func TestNewDeviationReporter(t *testing.T) {
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

	submissionPairs, err := getSubmissionPairs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}
	_, err = NewDeviationReporter(ctx, testItems.app.Host, testItems.app.Pubsub, submissionPairs)
	if err != nil {
		t.Fatalf("error creating new deviation reporter: %v", err)
	}
}

func TestStoreAndGetLastSubmission(t *testing.T) {
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

	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	aggregates, err := reporter.getLatestGlobalAggregates(ctx)
	if err != nil {
		t.Fatal("error getting latest global aggregates")
	}

	err = reporter.storeLastSubmission(ctx, aggregates)
	if err != nil {
		t.Fatal("error storing last submission")
	}

	loadedAggregates, err := reporter.getLastSubmission(ctx)
	if err != nil {
		t.Fatal("error getting last submission")
	}

	assert.EqualValues(t, aggregates, loadedAggregates)

}

func TestShouldReportDeviation(t *testing.T) {
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

	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}
	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
	if err != nil {
		t.Fatalf("error getting reporter: %v", err)
	}

	assert.False(t, reporter.shouldReportDeviation(0, 0))
	assert.True(t, reporter.shouldReportDeviation(0, 100000000))
	assert.False(t, reporter.shouldReportDeviation(100000000000, 100100000000))
	assert.True(t, reporter.shouldReportDeviation(100000000000, 105100000000))
	assert.False(t, reporter.shouldReportDeviation(100000000000, 0))
}

func TestGetDeviatingAggregates(t *testing.T) {
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

	submissionPairs, err := getSubmissionPairs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}
	reporter, err := NewDeviationReporter(ctx, testItems.app.Host, testItems.app.Pubsub, submissionPairs)
	if err != nil {
		t.Fatalf("error creating new deviation reporter: %v", err)
	}

	oldAggregates := []GlobalAggregate{{
		Name:  "test-aggregate",
		Value: 15,
		Round: 1,
	}}

	newAggregates := []GlobalAggregate{{
		Name:  "test-aggregate",
		Value: 30,
		Round: 2,
	}}

	result := reporter.getDeviatingAggregates(oldAggregates, newAggregates)
	assert.Equal(t, result, newAggregates)
}

func TestDeviationJob(t *testing.T) {
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

	submissionPairs, err := getSubmissionPairs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}
	reporter, err := NewDeviationReporter(ctx, testItems.app.Host, testItems.app.Pubsub, submissionPairs)
	if err != nil {
		t.Fatalf("error creating new deviation reporter: %v", err)
	}

	err = reporter.deviationJob()
	if err != nil {
		t.Fatal("error running deviation job")
	}
}
