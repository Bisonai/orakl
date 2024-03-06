package aggregator

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/libp2p"
	"github.com/gofiber/fiber/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	InsertLocalAggregateQuery  = `INSERT INTO local_aggregates (name, value, timestamp) VALUES (@name, @value, @time) RETURNING *;`
	RemoveGlobalAggregateQuery = `DELETE FROM global_aggregates;`
	RemoveLocalAggregateQuery  = `DELETE FROM local_aggregates;`
	DeleteAggregatorById       = `DELETE FROM aggregators;`
)

type TmpData struct {
	aggregator      Aggregator
	rLocalAggregate redisLocalAggregate
	pLocalAggregate pgsLocalAggregate
	globalAggregate globalAggregate
}

type TestItems struct {
	app         *App
	admin       *fiber.App
	host        *host.Host
	pubsub      *pubsub.PubSub
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

	app := New(mb)
	testItems.app = app

	h, err := libp2p.MakeHost(10001)
	if err != nil {
		return nil, nil, err
	}
	testItems.host = &h

	ps, err := libp2p.MakePubsub(ctx, h)
	if err != nil {
		return nil, nil, err
	}
	testItems.pubsub = ps

	testItems.topicString = "test-topic"

	tmpData, err := insertSampleData(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.tmpData = tmpData

	v1 := admin.Group("/api/v1")
	aggregator.Routes(v1)

	return aggregatorCleanup(ctx, admin, testItems), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpAggregator, err := db.QueryRow[Aggregator](ctx, aggregator.InsertAggregator, map[string]any{"name": "test_aggregator"})
	if err != nil {
		return nil, err
	}
	tmpData.aggregator = tmpAggregator

	localAggregateInsertTime := time.Now()

	key := "latestAggregate:test-aggregate"
	data, err := json.Marshal(redisLocalAggregate{Value: int64(10), Timestamp: localAggregateInsertTime})
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
	if err != nil {
		return nil, err
	}

	tmpData.rLocalAggregate = redisLocalAggregate{Value: int64(10), Timestamp: localAggregateInsertTime}

	tmpPLocalAggregate, err := db.QueryRow[pgsLocalAggregate](ctx, InsertLocalAggregateQuery, map[string]any{"name": "test-aggregate", "value": int64(10), "time": localAggregateInsertTime})
	if err != nil {
		return nil, err
	}
	tmpData.pLocalAggregate = tmpPLocalAggregate

	tmpGlobalAggregate, err := db.QueryRow[globalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"name": "test-aggregate", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate
	return tmpData, nil
}

func aggregatorCleanup(ctx context.Context, admin *fiber.App, testItems *TestItems) func() error {
	return func() error {
		err := db.QueryWithoutResult(ctx, RemoveGlobalAggregateQuery, nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, RemoveLocalAggregateQuery, nil)
		if err != nil {
			return err
		}

		err = db.Del(ctx, "latestAggregate:test-aggregate")
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, DeleteAggregatorById, nil)
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
	// zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()
	os.Exit(code)
}
