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

	configs, err := fetchConfigs()
	if err != nil {
		t.Fatalf("error getting submission pairs: %v", err)
	}
	groupedConfigs := groupConfigsBySubmitIntervals(configs)

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

	for groupInterval, pairs := range groupedConfigs {
		reporter, reporterErr := NewReporter(
			ctx,
			WithConfigs(pairs),
			WithInterval(groupInterval),
			WithContractAddress(contractAddress),
			WithCachedWhitelist(whitelist),
			WithKaiaHelper(tmpHelper),
			WithLatestDataMap(app.LatestDataMap),
			WithLatestSubmittedDataMap(app.LatestSubmittedDataMap),
		)
		if reporterErr != nil {
			t.Fatalf("error creating new reporter: %v", reporterErr)
		}
		app.Reporters = append(app.Reporters, reporter)
	}

	deviationReporter, errNewDeviationReporter := NewReporter(
		ctx,
		WithConfigs(configs),
		WithInterval(DEVIATION_INTERVAL),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(whitelist),
		WithJobType(DeviationJob),
		WithKaiaHelper(tmpHelper),
		WithLatestDataMap(app.LatestDataMap),
		WithLatestSubmittedDataMap(app.LatestSubmittedDataMap),
	)
	if errNewDeviationReporter != nil {
		if err != nil {
			t.Fatalf("error creating new deviation reporter: %v", err)
		}
	}
	app.Reporters = append(app.Reporters, deviationReporter)
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

	configs, err := fetchConfigs()
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
	deviationReporter, err := NewReporter(
		ctx,
		WithConfigs(configs),
		WithInterval(5000),
		WithContractAddress(contractAddress),
		WithCachedWhitelist(whitelist),
		WithJobType(DeviationJob),
		WithKaiaHelper(tmpHelper),
		WithLatestDataMap(app.LatestDataMap),
		WithLatestSubmittedDataMap(app.LatestSubmittedDataMap),
	)
	if err != nil {
		t.Fatalf("error creating new deviation reporter: %v", err)
	}

	for _, config := range configs {
		app.LatestSubmittedDataMap.Store(config.Name, int64(1))
		app.LatestDataMap.Store(config.Name, SubmissionData{
			Value: int64(2),
		})
	}

	deviatingAggregates := GetDeviatingAggregates(deviationReporter.LatestSubmittedDataMap, app.LatestDataMap, 0.05)
	assert.Equal(t, len(configs), len(deviatingAggregates))
}
