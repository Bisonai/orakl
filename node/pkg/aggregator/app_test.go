//nolint:all
package aggregator

import (
	"context"
	"strconv"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/admin/tests"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
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

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}
}

func TestGetAggregators(t *testing.T) {
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

	aggregators, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting aggregators")
	}

	assert.Equal(t, len(aggregators), 1)
	assert.Equal(t, aggregators[0].Name, testItems.tmpData.config.Name)
}

func TestStartAggregator(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}

		if aggregatorStopErr := testItems.app.stopAggregatorById(testItems.tmpData.config.ID); aggregatorStopErr != nil {
			t.Logf("error stopping aggregator: %v", aggregatorStopErr)
		}
	}()

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	err = testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.config.ID])
	if err != nil {
		t.Fatal("error starting aggregator")
	}

	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning)
}

func TestStartAggregatorById(t *testing.T) {
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

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	if testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning {
		t.Fatal("Aggregator should not be running before test")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	err = testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.config.ID].Config.ID)
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning, true)
}

func TestStopAggregator(t *testing.T) {
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

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	err = testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.config.ID])
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning)

	err = testItems.app.stopAggregator(testItems.app.Aggregators[testItems.tmpData.config.ID])
	if err != nil {
		t.Fatal("error stopping aggregator")
	}
	assert.Equal(t, false, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning)
}

func TestStopAggregatorById(t *testing.T) {
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

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	err = testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.config.ID])
	if err != nil {
		t.Fatal("error starting aggregator")
	}

	err = testItems.app.stopAggregatorById(testItems.app.Aggregators[testItems.tmpData.config.ID].Config.ID)
	if err != nil {
		t.Fatal("error stopping aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning, false)
}

func TestActivateAggregatorByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/activate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error activating aggregator: %v", err)
	}

	aggregator := testItems.app.Aggregators[testItems.tmpData.config.ID]

	assert.NotNil(t, aggregator)
	assert.True(t, aggregator.isRunning)
}

func TestDeactivateAggregatorByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	err = testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.config.ID].Config.ID)
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning, true)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/deactivate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error deactivating aggregator: %v", err)
	}

	aggregator := testItems.app.Aggregators[testItems.tmpData.config.ID]
	assert.NotNil(t, aggregator)
	assert.False(t, aggregator.isRunning)
}

func TestStartAppByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}
	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/start", nil)
	if err != nil {
		t.Fatalf("error starting app: %v", err)
	}

	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning)
}

func TestStopAppByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}
	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/start", nil)
	if err != nil {
		t.Fatalf("error starting app: %v", err)
	}
	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/stop", nil)
	if err != nil {
		t.Fatalf("error stopping app: %v", err)
	}

	assert.Equal(t, false, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning)
}

func TestRefreshAppByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	configs, err := testItems.app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}
	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	lengthBefore := len(testItems.app.Aggregators)

	testItems.app.LatestLocalAggregates.Store(testItems.tmpData.config.ID, &LocalAggregate{})

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/start", nil)
	if err != nil {
		t.Fatalf("error starting app: %v", err)
	}

	time.Sleep(time.Second)

	err = testItems.app.Configs.Set([]nodeconfig.Entry{sampleConfigEntry("test_pair"), sampleConfigEntry("test_pair_2")})
	if err != nil {
		t.Fatalf("error adding config: %v", err)
	}
	for _, config := range testItems.app.Configs.Configs() {
		testItems.app.LatestLocalAggregates.Store(config.ID, &LocalAggregate{})
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing app: %v", err)
	}

	assert.Greater(t, len(testItems.app.Aggregators), lengthBefore)
}
