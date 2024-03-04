package aggregator

import (
	"context"
	"encoding/json"
	"time"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	InsertLocalAggregateQuery  = `INSERT INTO local_aggregates (name, value, timestamp) VALUES (@name, @value, @time);`
	RemoveGlobalAggregateQuery = `DELETE FROM global_aggregates WHERE name = @name AND round = @round RETURNING *;`
	RemoveLocalAggregateQuery  = `DELETE FROM local_aggregates WHERE name = @name AND timestamp = @timestamp RETURNING *;`
	DeleteAggregatorById       = `DELETE FROM aggregators WHERE id = @id RETURNING *;`
)

type TmpData struct {
	aggregator      Aggregator
	rLocalAggregate redisLocalAggregate
	pLocalAggregate pgsLocalAggregate
	globalAggregate globalAggregate
}

type TestItems struct {
	app         *App
	host        *host.Host
	pubsub      *pubsub.PubSub
	topicString string
	messageBus  *bus.MessageBus
	aggregator  *App
	tmpData     *TmpData
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.New(10)
	testItems.messageBus = mb

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

	return aggregatorCleanup(ctx, testItems), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpAggregator, err := db.QueryRow[Aggregator](ctx, aggregator.InsertAggregator, map[string]any{"name": "test_aggregator"})
	if err != nil {
		return nil, err
	}
	tmpData.aggregator = tmpAggregator

	localAggregateInsertTime := time.Now()

	key := "latestAggregate:test"
	data, err := json.Marshal(redisLocalAggregate{Value: int64(10), Timestamp: localAggregateInsertTime})
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
	if err != nil {
		return nil, err
	}

	tmpData.rLocalAggregate = redisLocalAggregate{Value: int64(10), Timestamp: time.Now()}

	tmpPLocalAggregate, err := db.QueryRow[pgsLocalAggregate](ctx, InsertLocalAggregateQuery, map[string]any{"name": "test", "value": int64(10), "time": localAggregateInsertTime})
	if err != nil {
		return nil, err
	}
	tmpData.pLocalAggregate = tmpPLocalAggregate

	tmpGlobalAggregate, err := db.QueryRow[globalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"name": "test", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate
	return tmpData, nil
}

func aggregatorCleanup(ctx context.Context, testItems *TestItems) func() error {
	return func() error {
		_, err := db.QueryRow[Aggregator](ctx, RemoveGlobalAggregateQuery, map[string]any{"name": testItems.tmpData.globalAggregate.Name, "round": testItems.tmpData.globalAggregate.Round})
		if err != nil {
			return err
		}

		_, err = db.QueryRow[pgsLocalAggregate](ctx, RemoveLocalAggregateQuery, map[string]any{"name": testItems.tmpData.pLocalAggregate.Name, "timestamp": testItems.tmpData.pLocalAggregate.Timestamp})
		if err != nil {
			return err
		}

		err = db.Del(ctx, "latestAggregate:test")
		if err != nil {
			return err
		}

		_, err = db.QueryRow[Aggregator](ctx, aggregator.DeleteAggregatorById, map[string]any{"id": testItems.tmpData.aggregator.ID})
		if err != nil {
			return err
		}
		return nil
	}

}
