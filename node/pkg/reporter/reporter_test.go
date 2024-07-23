//nolint:all
package reporter

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/raft"
	"github.com/klaytn/klaytn/crypto"
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

	submissionPairs, err := getConfigs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}
	groupedSubmissionPairs := groupConfigsBySubmitIntervals(submissionPairs)

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		t.Fatal("SUBMISSION_PROXY_CONTRACT not set")
	}

	tmpHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Fatalf("error creating chain helper: %v", err)
	}
	defer tmpHelper.Close()

	whitelist, err := ReadOnchainWhitelist(ctx, tmpHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		t.Fatalf("error reading onchain whitelist: %v", err)
	}

	for groupInterval, pairs := range groupedSubmissionPairs {
		_, err := NewReporter(
			ctx,
			WithHost(testItems.app.Host),
			WithPubsub(testItems.app.Pubsub),
			WithConfigs(pairs),
			WithInterval(groupInterval),
			WithContractAddress(contractAddress),
			WithCachedWhitelist(whitelist),
		)
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

	err = reporter.handleCustomMessage(ctx, raft.Message{})
	assert.ErrorIs(t, err, errorSentinel.ErrReporterUnknownMessageType)

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
	result, err := GetLatestGlobalAggregates(ctx, reporter.SubmissionPairs)
	if err != nil {
		t.Fatal("error getting latest global aggregates")
	}

	assert.Equal(t, result[0].ConfigID, testItems.tmpData.globalAggregate.ConfigID)
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
		ConfigID: testItems.tmpData.config.ID,
		Value:    15,
		Round:    1,
	}}
	result := FilterInvalidAggregates(aggregates, reporter.SubmissionPairs)
	assert.Equal(t, result, aggregates)

	reporter.SubmissionPairs = map[int32]SubmissionPair{testItems.tmpData.config.ID: {LastSubmission: 1, Name: "test-aggregate"}}
	result = FilterInvalidAggregates(aggregates, reporter.SubmissionPairs)
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
		ConfigID: testItems.tmpData.config.ID,
		Value:    15,
		Round:    1,
	}
	result := IsAggValid(agg, reporter.SubmissionPairs)
	assert.Equal(t, result, true)

	reporter.SubmissionPairs = map[int32]SubmissionPair{testItems.tmpData.config.ID: {LastSubmission: 1, Name: "test-aggregate"}}
	result = IsAggValid(agg, reporter.SubmissionPairs)
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
		ConfigID:  testItems.tmpData.config.ID,
		Value:     15,
		Round:     1,
		Timestamp: testItems.tmpData.proofTime,
	}

	rawProofs, err := GetProofsRdb(ctx, []GlobalAggregate{agg})
	if err != nil {
		t.Fatal("error getting proofs")
	}

	proofMap := ProofsToMap(rawProofs)

	feedHashes, values, timestamps, proofs, err := MakeContractArgsWithProofs([]GlobalAggregate{agg}, reporter.SubmissionPairs, proofMap)
	if err != nil {
		t.Fatal("error making contract args")
	}
	assert.Equal(t, [32]byte(crypto.Keccak256([]byte(reporter.SubmissionPairs[agg.ConfigID].Name))), feedHashes[0])
	assert.Equal(t, big.NewInt(15), values[0])

	proofArr := make([][]byte, len(proofs))
	for i, p := range rawProofs {
		proofArr[i] = p.Proof
	}

	assert.EqualValues(t, proofs, proofArr)
	assert.Equal(t, testItems.tmpData.proofTime.Unix(), timestamps[0].Int64())

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

	result, err := GetLatestGlobalAggregatesRdb(ctx, reporter.SubmissionPairs)
	if err != nil {
		t.Fatal("error getting latest global aggregates from rdb")
	}

	assert.Equal(t, result[0].ConfigID, testItems.tmpData.globalAggregate.ConfigID)
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

	result, err := GetLatestGlobalAggregatesPgsql(ctx, reporter.SubmissionPairs)
	if err != nil {
		t.Fatal("error getting latest global aggregates from pgs")
	}

	assert.Equal(t, result[0].ConfigID, testItems.tmpData.globalAggregate.ConfigID)
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

	agg := testItems.tmpData.globalAggregate
	result, err := GetProofsRdb(ctx, []GlobalAggregate{agg})
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

	agg := testItems.tmpData.globalAggregate
	result, err := GetProofsPgsql(ctx, []GlobalAggregate{agg})
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

	submissionPairs, err := getConfigs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		t.Fatal("SUBMISSION_PROXY_CONTRACT not set")
	}

	tmpHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Fatalf("error creating chain helper: %v", err)
	}
	defer tmpHelper.Close()

	whitelist, err := ReadOnchainWhitelist(ctx, tmpHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		t.Fatalf("error reading onchain whitelist: %v", err)
	}
	_, err = NewReporter(
		ctx,
		WithHost(testItems.app.Host),
		WithPubsub(testItems.app.Pubsub),
		WithConfigs(submissionPairs),
		WithInterval(5000),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(whitelist),
		WithJobType(DeviationJob),
	)
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

	aggregates, err := GetLatestGlobalAggregates(ctx, reporter.SubmissionPairs)
	if err != nil {
		t.Fatal("error getting latest global aggregates")
	}

	err = StoreLastSubmission(ctx, aggregates)
	if err != nil {
		t.Fatal("error storing last submission")
	}

	loadedAggregates, err := GetLastSubmission(ctx, reporter.SubmissionPairs)
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

	assert.False(t, ShouldReportDeviation(0, 0, 0.05))
	assert.True(t, ShouldReportDeviation(0, 100000000, 0.05))
	assert.False(t, ShouldReportDeviation(100000000000, 100100000000, 0.05))
	assert.True(t, ShouldReportDeviation(100000000000, 105100000000, 0.05))
	assert.False(t, ShouldReportDeviation(100000000000, 0, 0.05))
}

func TestGetDeviationThreshold(t *testing.T) {
	ctx := context.Background()
	cleanup, _, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	assert.Equal(t, 0.05, GetDeviationThreshold(15*time.Second))
	assert.Equal(t, 0.01, GetDeviationThreshold(60*time.Minute))
	assert.Equal(t, 0.05, GetDeviationThreshold(1*time.Second))
	assert.Equal(t, 0.01, GetDeviationThreshold(2*time.Hour))
	assert.Less(t, GetDeviationThreshold(30*time.Minute), 0.05)
}

func TestGetDeviatingAggregates(t *testing.T) {
	ctx := context.Background()
	cleanup, _, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	oldAggregates := []GlobalAggregate{{
		ConfigID: 2,
		Value:    15,
		Round:    1,
	}}

	newAggregates := []GlobalAggregate{{
		ConfigID: 2,
		Value:    30,
		Round:    2,
	}}

	result := GetDeviatingAggregates(oldAggregates, newAggregates, 0.05)
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

	submissionPairs, err := getConfigs(ctx)
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}

	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddress == "" {
		t.Fatal("SUBMISSION_PROXY_CONTRACT not set")
	}

	tmpHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Fatalf("error creating chain helper: %v", err)
	}
	defer tmpHelper.Close()

	whitelist, err := ReadOnchainWhitelist(ctx, tmpHelper, contractAddress, GET_ONCHAIN_WHITELIST)
	if err != nil {
		t.Fatalf("error reading onchain whitelist: %v", err)
	}
	reporter, err := NewReporter(
		ctx,
		WithHost(testItems.app.Host),
		WithPubsub(testItems.app.Pubsub),
		WithConfigs(submissionPairs),
		WithInterval(5000),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(whitelist),
		WithJobType(DeviationJob),
	)
	if err != nil {
		t.Fatalf("error creating new deviation reporter: %v", err)
	}

	err = reporter.deviationJob()
	if err != nil {
		t.Fatal("error running deviation job")
	}
}
