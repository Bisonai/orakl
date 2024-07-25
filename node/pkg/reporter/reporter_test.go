//nolint:all
package reporter

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"github.com/stretchr/testify/assert"
)

func TestNewReporter(t *testing.T) {
	ctx := context.Background()
	app := New()

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
			WithConfigs(pairs),
			WithInterval(groupInterval),
			WithContractAddress(contractAddress),
			WithCachedWhitelist(whitelist),
			WithKaiaHelper(tmpHelper),
			WithLatestData(app.LatestData),
		)
		if err != nil {
			t.Fatalf("error creating new reporter: %v", err)
		}
	}
}

func TestShouldReportDeviation(t *testing.T) {
	ctx := context.Background()
	app := New()

	err := app.setReporters(ctx)
	if err != nil {
		t.Fatalf("error setting reporters: %v", err)
	}

	assert.False(t, ShouldReportDeviation(0, 0, 0.05))
	assert.True(t, ShouldReportDeviation(0, 100000000, 0.05))
	assert.False(t, ShouldReportDeviation(100000000000, 100100000000, 0.05))
	assert.True(t, ShouldReportDeviation(100000000000, 105100000000, 0.05))
	assert.False(t, ShouldReportDeviation(100000000000, 0, 0.05))
}

func TestGetDeviatingAggregates(t *testing.T) {
	ctx := context.Background()
	app := New()

	configs, err := getConfigs(ctx)
	if err != nil {
		t.Fatalf("error getting configs: %v", err)
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
		WithConfigs(configs),
		WithInterval(5000),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(whitelist),
		WithJobType(DeviationJob),
		WithKaiaHelper(tmpHelper),
		WithLatestData(app.LatestData),
	)
	if err != nil {
		t.Fatalf("error creating new deviation reporter: %v", err)
	}

	for _, config := range configs {
		pair := reporter.SubmissionPairs[config.ID]
		pair.LastSubmission = 1
		reporter.SubmissionPairs[config.ID] = pair

		app.LatestData.Store(config.Name, SubmissionData{
			Value: 2,
		})
	}

	deviatingAggregates := GetDeviatingAggregates(reporter.SubmissionPairs, app.LatestData, 0.05)
	assert.Equal(t, len(reporter.SubmissionPairs), len(deviatingAggregates))
}
