package aggregator

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/keys"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const (
	InsertConfigQuery         = `INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval) VALUES (@name, @fetch_interval, @aggregate_interval, @submit_interval) RETURNING name, id, aggregate_interval;`
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
	app         *App
	admin       *fiber.App
	topicString string
	messageBus  *bus.MessageBus
	tmpData     *TmpData
	signHelper  *helper.Signer
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.New(10)
	testItems.messageBus = mb

	admin, err := utils.Setup(utils.SetupInfo{
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

	tmpData, err := insertSampleData(ctx, app)
	if err != nil {
		return nil, nil, err
	}
	testItems.tmpData = tmpData

	signHelper, err := helper.NewSigner(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.signHelper = signHelper

	v1 := admin.Group("/api/v1")
	aggregator.Routes(v1)
	config.Routes(v1)

	return aggregatorCleanup(ctx, admin, app), testItems, nil
}

func insertSampleData(ctx context.Context, app *App) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": "test_pair", "fetch_interval": 2000, "aggregate_interval": 5000, "submit_interval": 15000})
	if err != nil {
		return nil, err
	}

	tmpData.config = tmpConfig

	localAggregateInsertTime := time.Now()

	key := keys.LocalAggregateKey(tmpConfig.ID)
	data, err := json.Marshal(LocalAggregate{ConfigID: tmpConfig.ID, Value: int64(10), Timestamp: localAggregateInsertTime})
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
	if err != nil {
		return nil, err
	}

	tmpData.rLocalAggregate = LocalAggregate{ConfigID: tmpConfig.ID, Value: int64(10), Timestamp: localAggregateInsertTime}

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

		for i := range app.Aggregators {
			err = db.Del(ctx, keys.LocalAggregateKey(i))
			if err != nil {
				return err
			}
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
