package aggregator

import (
	"context"
	"os"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/admin/aggregator"
	"bisonai.com/miko/node/pkg/admin/config"
	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/chain/helper"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/db"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const (
	InsertConfigQuery         = `INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval, decimals, feed_data_freshness) VALUES (@name, @fetch_interval, @aggregate_interval, @submit_interval, @decimals, @feed_data_freshness) RETURNING name, id, aggregate_interval;`
	InsertLocalAggregateQuery = `INSERT INTO local_aggregates (config_id, value, timestamp) VALUES (@config_id, @value, @time) RETURNING *;`
	DeleteGlobalAggregates    = `DELETE FROM global_aggregates;`
	DeleteLocalAggregates     = `DELETE FROM local_aggregates;`
	DeleteConfigs             = `DELETE FROM configs;`
)

type TmpData struct {
	config          Config
	rLocalAggregate LocalAggregate
	pLocalAggregate LocalAggregate
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

	app := New(mb, h, ps)
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

	v1 := admin.Group("/api/v1")
	aggregator.Routes(v1)
	config.Routes(v1)

	return aggregatorCleanup(ctx, admin, app), testItems, nil
}

func insertSampleData(ctx context.Context, app *App, latestLocalAggMap *LatestLocalAggregates) (*TmpData, error) {
	_ = db.QueryWithoutResult(ctx, DeleteConfigs, nil)

	var tmpData = new(TmpData)

	tmpConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": "test_pair", "fetch_interval": 2000, "aggregate_interval": 5000, "submit_interval": 15000, "decimals": 8, "feed_data_freshness": 1000})
	if err != nil {
		return nil, err
	}

	tmpData.config = tmpConfig

	localAggregateInsertTime := time.Now()

	tmpLocalAggregate := &LocalAggregate{ConfigID: tmpConfig.ID, Value: int64(10), Timestamp: localAggregateInsertTime}
	latestLocalAggMap.Store(tmpConfig.ID, tmpLocalAggregate)

	tmpData.rLocalAggregate = *tmpLocalAggregate

	tmpPLocalAggregate, err := db.QueryRow[LocalAggregate](ctx, InsertLocalAggregateQuery, map[string]any{"config_id": tmpConfig.ID, "value": int64(10), "time": localAggregateInsertTime})
	if err != nil {
		return nil, err
	}
	tmpData.pLocalAggregate = tmpPLocalAggregate

	globalAggregateInsertTime := time.Now()

	tmpGlobalAggregate, err := db.QueryRow[GlobalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"config_id": tmpConfig.ID, "value": int64(15), "round": int32(1), "timestamp": globalAggregateInsertTime})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate

	return tmpData, nil
}

func aggregatorCleanup(ctx context.Context, admin *fiber.App, app *App) func() error {
	return func() error {
		err := db.QueryWithoutResult(ctx, DeleteConfigs, nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, DeleteGlobalAggregates, nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, DeleteLocalAggregates, nil)
		if err != nil {
			return err
		}

		err = admin.Shutdown()
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
