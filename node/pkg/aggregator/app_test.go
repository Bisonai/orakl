//nolint:all
package aggregator

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/tests"
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	if testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning {
		t.Fatal("Aggregator should not be running before test")
	}

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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

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

func TestGetAggregatorByName(t *testing.T) {
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	aggregator, err := testItems.app.getAggregatorByName(testItems.tmpData.config.Name)
	if err != nil {
		t.Fatal("error getting aggregator by name")
	}
	assert.NotNil(t, aggregator)
	assert.Equal(t, aggregator.Name, testItems.tmpData.config.Name)
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/activate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error activating aggregator: %v", err)
	}

	aggregator, err := testItems.app.getAggregatorByName(testItems.tmpData.config.Name)
	if err != nil {
		t.Fatal("error getting aggregator by name")
	}
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	err = testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.config.ID].Config.ID)
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.config.ID].isRunning, true)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/deactivate/"+strconv.Itoa(int(testItems.tmpData.config.ID)), nil)
	if err != nil {
		t.Fatalf("error deactivating aggregator: %v", err)
	}

	aggregator, err := testItems.app.getAggregatorByName(testItems.tmpData.config.Name)
	if err != nil {
		t.Fatal("error getting aggregator by name")
	}
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	lengthBefore := len(testItems.app.Aggregators)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/start", nil)
	if err != nil {
		t.Fatalf("error starting app: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/config", map[string]any{"name": "test_pair_2", "fetch_interval": 2000, "aggregate_interval": 5000, "submit_interval": 15000})
	if err != nil {
		t.Fatalf("error creating new aggregator: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing app: %v", err)
	}

	assert.Greater(t, len(testItems.app.Aggregators), lengthBefore)
}
