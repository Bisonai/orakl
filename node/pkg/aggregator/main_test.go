package aggregator

import (
	"context"
	"os"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/admin/aggregator"
	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/chain/helper"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/db"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/nodeconfig"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type TmpData struct {
	config          Config
	rLocalAggregate LocalAggregate
	globalAggregate GlobalAggregate
}

type TestItems struct {
	app               *App
	admin             *fiber.App
	topicString       string
	messageBus        *bus.MessageBus
	tmpData           *TmpData
	signer            *helper.Signer
	latestLocalAggMap *LatestLocalAggregates
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.New(10)
	testItems.messageBus = mb

	admin, err := utils.Setup(ctx, utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})
	if err != nil {
		return nil, nil, err
	}
	testItems.admin = admin

	h, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch())
	if err != nil {
		return nil, nil, err
	}

	ps, err := libp2pSetup.MakePubsub(ctx, h)
	if err != nil {
		return nil, nil, err
	}

	app := New(mb, h, ps, nil)
	app.Configs = nodeconfig.New()
	testItems.app = app

	testItems.topicString = "test-topic"
	testItems.latestLocalAggMap = NewLatestLocalAggregates()
	tmpData, err := insertSampleData(ctx, app, testItems.latestLocalAggMap)
	if err != nil {
		return nil, nil, err
	}
	testItems.tmpData = tmpData

	// Static mode (WithSignerPk): the aggregator tests only need a working signer that produces
	// proofs; static mode bypasses the on-chain whitelist gate so they don't depend on live chain
	// state (the managed reconcile/rotate path is covered by helper unit + integration tests).
	signHelper, err := helper.NewSigner(ctx, helper.WithSignerPk("27894b84849f129e08f37634be4e8ccc4c7267d824eb8cfd285185854ba5b78d"))
	if err != nil {
		return nil, nil, err
	}
	testItems.signer = signHelper
	// the app has no state store, so it must not build its own managed signer
	app.Signer = signHelper

	v1 := admin.Group("/api/v1")
	aggregator.Routes(v1)

	return aggregatorCleanup(ctx, admin, app), testItems, nil
}

func insertSampleData(ctx context.Context, app *App, latestLocalAggMap *LatestLocalAggregates) (*TmpData, error) {
	var tmpData = new(TmpData)

	if err := app.Configs.Set([]nodeconfig.Entry{sampleConfigEntry("test_pair")}); err != nil {
		return nil, err
	}
	tmpConfig := app.Configs.Configs()[0]

	tmpData.config = tmpConfig

	localAggregateInsertTime := time.Now()

	tmpLocalAggregate := &LocalAggregate{ConfigID: tmpConfig.ID, Value: int64(10), Timestamp: localAggregateInsertTime}
	latestLocalAggMap.Store(tmpConfig.ID, tmpLocalAggregate)

	tmpData.rLocalAggregate = *tmpLocalAggregate
	tmpData.globalAggregate = GlobalAggregate{ConfigID: tmpConfig.ID, Value: int64(15), Round: int32(1), Timestamp: time.Now()}

	return tmpData, nil
}

func sampleConfigEntry(name string) nodeconfig.Entry {
	fetchInterval, aggregateInterval, submitInterval := int32(2000), int32(5000), int32(15000)
	decimals, freshness := 8, 1000
	return nodeconfig.Entry{
		Name:              name,
		FetchInterval:     &fetchInterval,
		AggregateInterval: &aggregateInterval,
		SubmitInterval:    &submitInterval,
		Decimals:          &decimals,
		FeedDataFreshness: &freshness,
	}
}

func aggregatorCleanup(ctx context.Context, admin *fiber.App, app *App) func() error {
	return func() error {
		err := admin.Shutdown()
		if err != nil {
			return err
		}

		err = app.stopAllAggregators()
		if err != nil {
			return err
		}

		return app.Host.Close()
	}

}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()
	os.Exit(code)
}
