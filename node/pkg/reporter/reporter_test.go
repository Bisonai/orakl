//nolint:all
package reporter

// func TestNewReporter(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, testItems, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	submissionPairs, err := getConfigs(ctx)
// 	if err != nil {
// 		t.Fatalf("error getting submission pairs: %v", err)
// 	}
// 	groupedSubmissionPairs := groupConfigsBySubmitIntervals(submissionPairs)

// 	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
// 	if contractAddress == "" {
// 		t.Fatal("SUBMISSION_PROXY_CONTRACT not set")
// 	}

// 	tmpHelper, err := helper.NewChainHelper(ctx)
// 	if err != nil {
// 		t.Fatalf("error creating chain helper: %v", err)
// 	}
// 	defer tmpHelper.Close()

// 	whitelist, err := ReadOnchainWhitelist(ctx, tmpHelper, contractAddress, GET_ONCHAIN_WHITELIST)
// 	if err != nil {
// 		t.Fatalf("error reading onchain whitelist: %v", err)
// 	}

// 	for groupInterval, pairs := range groupedSubmissionPairs {
// 		_, err := NewReporter(
// 			ctx,
// 			WithConfigs(pairs),
// 			WithInterval(groupInterval),
// 			WithContractAddress(contractAddress),
// 			WithCachedWhitelist(whitelist),
// 			WithKaiaHelper(tmpHelper),
// 			WithLatestData(testItems.app.LatestData),
// 		)
// 		if err != nil {
// 			t.Fatalf("error creating new reporter: %v", err)
// 		}
// 	}
// }

// func TestNewDeviationReporter(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, testItems, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	submissionPairs, err := getConfigs(ctx)
// 	if err != nil {
// 		t.Fatalf("error getting submission pairs: %v", err)
// 	}

// 	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
// 	if contractAddress == "" {
// 		t.Fatal("SUBMISSION_PROXY_CONTRACT not set")
// 	}

// 	tmpHelper, err := helper.NewChainHelper(ctx)
// 	if err != nil {
// 		t.Fatalf("error creating chain helper: %v", err)
// 	}
// 	defer tmpHelper.Close()

// 	whitelist, err := ReadOnchainWhitelist(ctx, tmpHelper, contractAddress, GET_ONCHAIN_WHITELIST)
// 	if err != nil {
// 		t.Fatalf("error reading onchain whitelist: %v", err)
// 	}
// 	_, err = NewReporter(
// 		ctx,
// 		WithConfigs(submissionPairs),
// 		WithInterval(5000),
// 		WithContractAddress(contractAddress),
// 		WithCachedWhitelist(whitelist),
// 		WithJobType(DeviationJob),
// 		WithKaiaHelper(tmpHelper),
// 		WithLatestData(testItems.app.LatestData),
// 	)
// 	if err != nil {
// 		t.Fatalf("error creating new deviation reporter: %v", err)
// 	}
// }

// func TestStoreAndGetLastSubmission(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, testItems, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
// 	if err != nil {
// 		t.Fatalf("error setting reporters: %v", err)
// 	}
// 	reporter, err := testItems.app.GetReporterWithInterval(TestInterval)
// 	if err != nil {
// 		t.Fatalf("error getting reporter: %v", err)
// 	}

// 	aggregates, err := GetLatestGlobalAggregates(ctx, reporter.SubmissionPairs)
// 	if err != nil {
// 		t.Fatal("error getting latest global aggregates")
// 	}

// 	err = StoreLastSubmission(ctx, aggregates)
// 	if err != nil {
// 		t.Fatal("error storing last submission")
// 	}

// 	loadedAggregates, err := GetLastSubmission(ctx, reporter.SubmissionPairs)
// 	if err != nil {
// 		t.Fatal("error getting last submission")
// 	}

// 	assert.EqualValues(t, aggregates, loadedAggregates)

// }

// func TestShouldReportDeviation(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, testItems, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	err = testItems.app.setReporters(ctx, testItems.app.Host, testItems.app.Pubsub)
// 	if err != nil {
// 		t.Fatalf("error setting reporters: %v", err)
// 	}

// 	assert.False(t, ShouldReportDeviation(0, 0, 0.05))
// 	assert.True(t, ShouldReportDeviation(0, 100000000, 0.05))
// 	assert.False(t, ShouldReportDeviation(100000000000, 100100000000, 0.05))
// 	assert.True(t, ShouldReportDeviation(100000000000, 105100000000, 0.05))
// 	assert.False(t, ShouldReportDeviation(100000000000, 0, 0.05))
// }

// func TestGetDeviationThreshold(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, _, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	assert.Equal(t, 0.05, GetDeviationThreshold(15*time.Second))
// 	assert.Equal(t, 0.01, GetDeviationThreshold(60*time.Minute))
// 	assert.Equal(t, 0.05, GetDeviationThreshold(1*time.Second))
// 	assert.Equal(t, 0.01, GetDeviationThreshold(2*time.Hour))
// 	assert.Less(t, GetDeviationThreshold(30*time.Minute), 0.05)
// }

// func TestGetDeviatingAggregates(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, _, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	oldAggregates := []GlobalAggregate{{
// 		ConfigID: 2,
// 		Value:    15,
// 		Round:    1,
// 	}}

// 	newAggregates := []GlobalAggregate{{
// 		ConfigID: 2,
// 		Value:    30,
// 		Round:    2,
// 	}}

// 	result := GetDeviatingAggregates(oldAggregates, newAggregates, 0.05)
// 	assert.Equal(t, result, newAggregates)
// }

// func TestDeviationJob(t *testing.T) {
// 	ctx := context.Background()
// 	cleanup, testItems, err := setup(ctx)
// 	if err != nil {
// 		t.Fatalf("error setting up test: %v", err)
// 	}
// 	defer func() {
// 		if cleanupErr := cleanup(); cleanupErr != nil {
// 			t.Logf("Cleanup failed: %v", cleanupErr)
// 		}
// 	}()

// 	submissionPairs, err := getConfigs(ctx)
// 	if err != nil {
// 		t.Fatalf("error getting submission pairs: %v", err)
// 	}

// 	contractAddress := os.Getenv("SUBMISSION_PROXY_CONTRACT")
// 	if contractAddress == "" {
// 		t.Fatal("SUBMISSION_PROXY_CONTRACT not set")
// 	}

// 	tmpHelper, err := helper.NewChainHelper(ctx)
// 	if err != nil {
// 		t.Fatalf("error creating chain helper: %v", err)
// 	}
// 	defer tmpHelper.Close()

// 	whitelist, err := ReadOnchainWhitelist(ctx, tmpHelper, contractAddress, GET_ONCHAIN_WHITELIST)
// 	if err != nil {
// 		t.Fatalf("error reading onchain whitelist: %v", err)
// 	}
// 	reporter, err := NewReporter(
// 		ctx,
// 		WithHost(testItems.app.Host),
// 		WithPubsub(testItems.app.Pubsub),
// 		WithConfigs(submissionPairs),
// 		WithInterval(5000),
// 		WithContractAddress(contractAddress),
// 		WithCachedWhitelist(whitelist),
// 		WithJobType(DeviationJob),
// 	)
// 	if err != nil {
// 		t.Fatalf("error creating new deviation reporter: %v", err)
// 	}

// 	err = reporter.deviationJob()
// 	if err != nil {
// 		t.Fatal("error running deviation job")
// 	}
// }
