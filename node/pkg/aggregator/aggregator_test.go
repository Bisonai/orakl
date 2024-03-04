//nolint:all
package aggregator

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

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
	defer cleanup()

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
	if err != nil {
		fmt.Println(err)
		t.Fatal("error initializing app")
	}
}

func TestGetAggregators(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

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
	defer cleanup()

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	go testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, true, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning)
}

func TestStartAggregatorById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	go testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].Aggregator.ID)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning, true)
}

func TestStopAggregator(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	go testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	time.Sleep(100 * time.Millisecond)

	err = testItems.app.stopAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	if err != nil {
		t.Fatal("error stopping aggregator")
	}
	assert.Equal(t, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].isRunning, false)
}

func TestStopAggregatorById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	go testItems.app.startAggregator(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID])
	time.Sleep(100 * time.Millisecond)

	err = testItems.app.stopAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].Aggregator.ID)
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
	defer cleanup()

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
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
	defer cleanup()

	testItems.app.subscribe(ctx)

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
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
	defer cleanup()

	testItems.app.subscribe(ctx)

	err = testItems.app.initialize(ctx, *testItems.host, testItems.pubsub)
	if err != nil {
		t.Fatal("error initializing app")
	}

	go testItems.app.startAggregatorById(ctx, testItems.app.Aggregators[testItems.tmpData.aggregator.ID].Aggregator.ID)
	time.Sleep(100 * time.Millisecond)
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
