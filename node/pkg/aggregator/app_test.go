//nolint:all
package aggregator

import (
	"context"
	"strconv"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/admin/tests"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/chain/helper"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"bisonai.com/miko/node/pkg/raft"
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

// newLifecycleTestApp builds an app on a real libp2p host/pubsub without the
// admin API (no Redis needed); lifecycle tests drive the app directly.
func newLifecycleTestApp(t *testing.T, ctx context.Context) (*App, int32) {
	t.Helper()
	h, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch())
	if err != nil {
		t.Fatalf("error creating host: %v", err)
	}
	ps, err := libp2pSetup.MakePubsub(ctx, h)
	if err != nil {
		t.Fatalf("error creating pubsub: %v", err)
	}
	app := New(bus.New(10), h, ps, nil)
	app.Configs = nodeconfig.New()
	if err = app.Configs.Set([]nodeconfig.Entry{sampleConfigEntry("test_pair")}); err != nil {
		t.Fatalf("error setting configs: %v", err)
	}
	signHelper, err := helper.NewSigner(ctx, helper.WithSignerPk("27894b84849f129e08f37634be4e8ccc4c7267d824eb8cfd285185854ba5b78d"))
	if err != nil {
		t.Fatalf("error creating signer: %v", err)
	}
	app.Signer = signHelper
	t.Cleanup(func() {
		_ = app.stopAllAggregators()
		_ = h.Close()
	})
	return app, app.Configs.Configs()[0].ID
}

// TestRestartAggregatorKeepsTopic: stop then start (deactivate→activate,
// stop→start app) must leave the aggregator on a usable topic. Stopping used
// to close the topic, so the restarted raft loop could not subscribe/publish.
func TestRestartAggregatorKeepsTopic(t *testing.T) {
	ctx := context.Background()
	app, configID := newLifecycleTestApp(t, ctx)

	configs, err := app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}
	err = app.setAggregators(ctx, app.Host, app.Pubsub, configs)
	if err != nil {
		t.Fatal("error initializing app")
	}

	agg := app.Aggregators[configID]
	topicName := agg.Raft.Topic.String()
	subscribed := func() bool {
		for _, name := range app.Pubsub.GetTopics() {
			if name == topicName {
				return true
			}
		}
		return false
	}

	if err = app.startAggregator(ctx, agg); err != nil {
		t.Fatal("error starting aggregator")
	}
	assert.Eventually(t, subscribed, 2*time.Second, 20*time.Millisecond, "not subscribed after first start")

	if err = app.stopAggregator(agg); err != nil {
		t.Fatal("error stopping aggregator")
	}
	// operator stop/start calls are not back to back
	time.Sleep(200 * time.Millisecond)

	if err = app.startAggregator(ctx, agg); err != nil {
		t.Fatal("error restarting aggregator")
	}
	assert.Eventually(t, subscribed, 2*time.Second, 20*time.Millisecond, "not subscribed after restart")
	assert.NoError(t, agg.Raft.Topic.Publish(ctx, []byte("ping")), "topic unusable after restart")
}

// TestRestartedLeaderLeadsAgain: a node stopped while leading must resign, so
// after a restart it can campaign and lead again. Otherwise it keeps the
// Leader role with no leader loop: it sends nothing, never campaigns and
// refuses votes; a lone node (kairos) stays stuck until a refresh.
func TestRestartedLeaderLeadsAgain(t *testing.T) {
	ctx := context.Background()
	app, configID := newLifecycleTestApp(t, ctx)

	configs, err := app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}
	if err = app.setAggregators(ctx, app.Host, app.Pubsub, configs); err != nil {
		t.Fatal("error initializing app")
	}
	agg := app.Aggregators[configID]
	roundID := func() int32 {
		agg.mu.RLock()
		defer agg.mu.RUnlock()
		return agg.RoundID
	}

	if err = app.startAggregator(ctx, agg); err != nil {
		t.Fatal("error starting aggregator")
	}
	isLeader := func() bool { return agg.Raft.GetRole() == raft.Leader }
	assert.Eventually(t, isLeader, 15*time.Second, 50*time.Millisecond, "lone node never led")
	firstTerm := agg.Raft.GetCurrentTerm()

	if err = app.stopAggregator(agg); err != nil {
		t.Fatal("error stopping aggregator")
	}
	assert.NotEqual(t, raft.Leader, agg.Raft.GetRole(), "stopped node still leader")

	if err = app.startAggregator(ctx, agg); err != nil {
		t.Fatal("error restarting aggregator")
	}
	assert.Eventually(t, func() bool {
		return isLeader() && agg.Raft.GetCurrentTerm() > firstTerm
	}, 15*time.Second, 50*time.Millisecond, "restarted node never led again")
	// leader loop runs again: rounds advance
	afterElection := roundID()
	assert.Eventually(t, func() bool { return roundID() > afterElection }, 15*time.Second, 50*time.Millisecond, "no rounds after restart")
}

// TestRefreshAfterStopRejoinsTopic: refresh tears aggregators down and joins
// the same topic again; that join must succeed without waiting on an
// asynchronous close.
func TestRefreshAfterStopRejoinsTopic(t *testing.T) {
	ctx := context.Background()
	app, configID := newLifecycleTestApp(t, ctx)

	configs, err := app.getConfigs(ctx)
	if err != nil {
		t.Fatal("error getting configs")
	}
	if err = app.setAggregators(ctx, app.Host, app.Pubsub, configs); err != nil {
		t.Fatal("error initializing app")
	}
	if err = app.startAllAggregators(ctx); err != nil {
		t.Fatal("error starting aggregators")
	}

	if err = app.setAggregators(ctx, app.Host, app.Pubsub, configs); err != nil {
		t.Fatalf("refresh failed to rejoin topic: %v", err)
	}
	if err = app.startAllAggregators(ctx); err != nil {
		t.Fatal("error restarting aggregators")
	}
	agg := app.Aggregators[configID]
	assert.NoError(t, agg.Raft.Topic.Publish(ctx, []byte("ping")), "topic unusable after refresh")
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
