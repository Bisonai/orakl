//nolint:all
package aggregator

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
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
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
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
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	aggregators, err := testItems.app.getAggregators(ctx)
	if err != nil {
		t.Fatal("error getting aggregators")
	}

	assert.Equal(t, len(aggregators), 1)
	assert.Equal(t, aggregators[0].Name, testItems.tmpData.aggregator.Name)
}

func TestStartAggregator(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}

		if err := testItems.app.stopAggregatorById(ctx, testItems.tmpData.aggregator.ID); err != nil {
			t.Logf("error stopping aggregator: %v", err)
		}
	}()

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	err = testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	if err != nil {
		t.Fatal("error starting aggregator")
	}

	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)
}

func TestStartAggregatorById(t *testing.T) {
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	if testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning {
		t.Fatal("Aggregator should not be running before test")
	}

	err = testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].AggregatorModel.ID)
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning, true)
}

func TestStopAggregator(t *testing.T) {
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	err = testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)

	err = testItems.app.stopAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	if err != nil {
		t.Fatal("error stopping aggregator")
	}
	assert.Equal(t, false, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)
}

func TestStopAggregatorById(t *testing.T) {
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	err = testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	if err != nil {
		t.Fatal("error starting aggregator")
	}

	err = testItems.app.stopAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].AggregatorModel.ID)
	if err != nil {
		t.Fatal("error stopping aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning, false)
}

func TestGetAggregatorByName(t *testing.T) {
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

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	aggregator, err := testItems.app.getAggregatorByName(testItems.tmpData.aggregator.Name)
	if err != nil {
		t.Fatal("error getting aggregator by name")
	}
	assert.NotNil(t, aggregator)
	assert.Equal(t, aggregator.Name, testItems.tmpData.aggregator.Name)
}

func TestActivateAggregatorByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	result, err := tests.PostRequest[aggregator.AggregatorModel](testItems.admin, "/api/v1/aggregator/activate/"+strconv.FormatInt(testItems.tmpData.aggregator.ID, 10), nil)
	if err != nil {
		t.Fatalf("error activating aggregator: %v", err)
	}

	assert.Equal(t, *result.Id, testItems.tmpData.aggregator.ID)

	aggregator, err := testItems.app.getAggregatorByName(testItems.tmpData.aggregator.Name)
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
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
		}
	}()

	testItems.app.subscribe(ctx)

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	err = testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].AggregatorModel.ID)
	if err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning, true)

	result, err := tests.PostRequest[aggregator.AggregatorModel](testItems.admin, "/api/v1/aggregator/deactivate/"+strconv.FormatInt(testItems.tmpData.aggregator.ID, 10), nil)
	if err != nil {
		t.Fatalf("error deactivating aggregator: %v", err)
	}

	assert.Equal(t, *result.Id, testItems.tmpData.aggregator.ID)

	aggregator, err := testItems.app.getAggregatorByName(testItems.tmpData.aggregator.Name)
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
		if err := cleanup(); err != nil {
			t.Logf("Cleanup failed: %v", err)
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

	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)
}

func TestStopAppByAdmin(t *testing.T) {
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

	testItems.app.subscribe(ctx)

	err = testItems.app.setAggregators(ctx, testItems.app.Host, testItems.app.Pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/start", nil)
	if err != nil {
		t.Fatalf("error starting app: %v", err)
	}
	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/stop", nil)
	if err != nil {
		t.Fatalf("error stopping app: %v", err)
	}

	assert.Equal(t, false, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)
}

func TestRefreshAppByAdmin(t *testing.T) {
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

	tmpAggregator, err := tests.PostRequest[AggregatorModel](testItems.admin, "/api/v1/aggregator", map[string]any{"name": "test_pair_2"})
	if err != nil {
		t.Fatalf("error creating new aggregator: %v", err)
	}

	_, err = tests.RawPostRequest(testItems.admin, "/api/v1/aggregator/refresh", nil)
	if err != nil {
		t.Fatalf("error refreshing app: %v", err)
	}

	assert.Greater(t, len(testItems.app.Aggregators), lengthBefore)

	//cleanup
	_, err = tests.DeleteRequest[AggregatorModel](testItems.admin, "/api/v1/aggregator/"+strconv.FormatInt(tmpAggregator.ID, 10), nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}
