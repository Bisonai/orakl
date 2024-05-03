package aggregator

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/utils"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const (
	InsertConfigQuery         = `INSERT INTO configs (name, address, fetch_interval, aggregate_interval, submit_interval) VALUES (@name, @address, @fetch_interval, @aggregate_interval, @submit_interval) RETURNING name, id, aggregate_interval;`
	InsertLocalAggregateQuery = `INSERT INTO local_aggregates (config_id, value, timestamp) VALUES (@config_id, @value, @time) RETURNING *;`
	DeleteGlobalAggregates    = `DELETE FROM global_aggregates;`
	DeleteLocalAggregates     = `DELETE FROM local_aggregates;`
	DeleteConfigs             = `DELETE FROM configs;`
)

type TmpData struct {
	config          Config
	rLocalAggregate LocalAggregate
	pLocalAggregate PgsLocalAggregate
	globalAggregate GlobalAggregate
}

type TestItems struct {
	app         *App
	admin       *fiber.App
	topicString string
	messageBus  *bus.MessageBus
	tmpData     *TmpData
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

	h, err := libp2p_setup.MakeHost(10001)
	if err != nil {
		return nil, nil, err
	}

	ps, err := libp2p_setup.MakePubsub(ctx, h)
	if err != nil {
		return nil, nil, err
	}

	app := New(mb, h, ps)
	testItems.app = app

	testItems.topicString = "test-topic"

	tmpData, err := insertSampleData(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.tmpData = tmpData

	v1 := admin.Group("/api/v1")
	aggregator.Routes(v1)
	config.Routes(v1)

	return aggregatorCleanup(ctx, admin), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": "test_pair", "address": "test_address", "fetch_interval": 2000, "aggregate_interval": 5000, "submit_interval": 15000})
	if err != nil {
		return nil, err
	}

	tmpData.config = tmpConfig

	localAggregateInsertTime := time.Now()

	key := "localAggregate:" + strconv.Itoa(int(tmpConfig.ID))
	data, err := json.Marshal(LocalAggregate{ConfigId: tmpConfig.ID, Value: int64(10), Timestamp: localAggregateInsertTime})
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
	if err != nil {
		return nil, err
	}

	tmpData.rLocalAggregate = LocalAggregate{ConfigId: tmpConfig.ID, Value: int64(10), Timestamp: localAggregateInsertTime}

	tmpPLocalAggregate, err := db.QueryRow[PgsLocalAggregate](ctx, InsertLocalAggregateQuery, map[string]any{"config_id": tmpConfig.ID, "value": int64(10), "time": localAggregateInsertTime})
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

func aggregatorCleanup(ctx context.Context, admin *fiber.App) func() error {
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

		err = db.Del(ctx, "localAggregate:test_pair")
		if err != nil {
			return err
		}

		err = admin.Shutdown()
		if err != nil {
			return err
		}
		return nil
	}

}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()
	os.Exit(code)
}
