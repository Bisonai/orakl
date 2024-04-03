package aggregator

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const (
	InsertLocalAggregateQuery = `INSERT INTO local_aggregates (name, value, timestamp) VALUES (@name, @value, @time) RETURNING *;`
	DeleteGlobalAggregates    = `DELETE FROM global_aggregates;`
	DeleteLocalAggregates     = `DELETE FROM local_aggregates;`
	DeleteAggregators         = `DELETE FROM aggregators;`
	DeleteAdapters            = `DELETE FROM adapters;`
)

type TmpData struct {
	aggregator      AggregatorModel
	rLocalAggregate redisLocalAggregate
	pLocalAggregate pgsLocalAggregate
	globalAggregate globalAggregate
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

	return aggregatorCleanup(ctx, admin), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	err := db.QueryWithoutResult(ctx, adapter.InsertAdapter, map[string]any{"name": "test_pair"})
	if err != nil {
		return nil, err
	}

	tmpAggregator, err := db.QueryRow[AggregatorModel](ctx, aggregator.InsertAggregator, map[string]any{"name": "test_pair"})
	if err != nil {
		return nil, err
	}

	tmpData.aggregator = tmpAggregator

	localAggregateInsertTime := time.Now()

	key := "localAggregate:test_pair"
	data, err := json.Marshal(redisLocalAggregate{Value: int64(10), Timestamp: localAggregateInsertTime})
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
	if err != nil {
		return nil, err
	}

	tmpData.rLocalAggregate = redisLocalAggregate{Value: int64(10), Timestamp: localAggregateInsertTime}

	tmpPLocalAggregate, err := db.QueryRow[pgsLocalAggregate](ctx, InsertLocalAggregateQuery, map[string]any{"name": "test_pair", "value": int64(10), "time": localAggregateInsertTime})
	if err != nil {
		return nil, err
	}
	tmpData.pLocalAggregate = tmpPLocalAggregate

	tmpGlobalAggregate, err := db.QueryRow[globalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"name": "test_pair", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate
	return tmpData, nil
}

func aggregatorCleanup(ctx context.Context, admin *fiber.App) func() error {
	return func() error {
		err := db.QueryWithoutResult(ctx, DeleteAggregators, nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, DeleteAdapters, nil)
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
