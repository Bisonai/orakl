package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"time"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const LEADER_TIMEOUT = 5 * time.Second

func NewNode(h host.Host, ps *pubsub.PubSub, topicString string) (*Aggregator, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	aggregator := Aggregator{
		Raft:            raft.NewRaftNode(h, ps, topic, 100, LEADER_TIMEOUT),
		CollectedPrices: map[int64][]int64{},
		AggregatorMutex: sync.Mutex{},
	}
	aggregator.Raft.LeaderJob = aggregator.LeaderJob
	aggregator.Raft.HandleCustomMessage = aggregator.HandleCustomMessage

	return &aggregator, nil
}

func (n *Aggregator) Run(ctx context.Context) {
	latestRoundId, err := n.getLatestRoundId(ctx)
	if err == nil && latestRoundId > 0 {
		n.RoundID = latestRoundId
	}

	n.Raft.Run(ctx)
}

func (n *Aggregator) LeaderJob() error {
	n.RoundID++
	n.Raft.IncreaseTerm()
	roundMessage := RoundSyncMessage{
		LeaderID: n.Raft.Host.ID().String(),
		RoundID:  n.RoundID,
	}

	marshalledRoundMessage, err := json.Marshal(roundMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     RoundSync,
		SentFrom: n.Raft.Host.ID().String(),
		Data:     json.RawMessage(marshalledRoundMessage),
	}

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) HandleCustomMessage(message raft.Message) error {
	switch message.Type {
	case RoundSync:
		return n.HandleRoundSyncMessage(message)
	case PriceData:
		return n.HandlePriceDataMessage(message)
	default:
		return errors.New("unknown message type")
	}
}

func (n *Aggregator) HandleRoundSyncMessage(msg raft.Message) error {
	var roundSyncMessage RoundSyncMessage
	err := json.Unmarshal(msg.Data, &roundSyncMessage)
	if err != nil {
		return err
	}

	if n.Raft.GetRole() != raft.Leader {
		n.RoundID = roundSyncMessage.RoundID
	}
	var updateValue int64 = -1
	value, updateTime, err := n.getLatestLocalAggregate(n.nodeCtx)

	if err == nil && n.LastLocalAggregateTime.IsZero() || !n.LastLocalAggregateTime.Equal(updateTime) {
		updateValue = value
		n.LastLocalAggregateTime = updateTime
	}

	priceDataMessage := PriceDataMessage{
		RoundID:   n.RoundID,
		PriceData: updateValue,
	}

	marshalledPriceDataMessage, err := json.Marshal(priceDataMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     PriceData,
		SentFrom: n.Raft.Host.ID().String(),
		Data:     json.RawMessage(marshalledPriceDataMessage),
	}

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) HandlePriceDataMessage(msg raft.Message) error {
	var priceDataMessage PriceDataMessage
	err := json.Unmarshal(msg.Data, &priceDataMessage)
	if err != nil {
		return err
	}
	n.AggregatorMutex.Lock()
	defer n.AggregatorMutex.Unlock()
	if _, ok := n.CollectedPrices[priceDataMessage.RoundID]; !ok {
		n.CollectedPrices[priceDataMessage.RoundID] = []int64{}
	}

	n.CollectedPrices[priceDataMessage.RoundID] = append(n.CollectedPrices[priceDataMessage.RoundID], priceDataMessage.PriceData)
	if len(n.CollectedPrices[priceDataMessage.RoundID]) >= n.Raft.SubscribersCount()+1 {
		defer delete(n.CollectedPrices, priceDataMessage.RoundID)
		filteredCollectedPrices := FilterNegative(n.CollectedPrices[priceDataMessage.RoundID])

		median, err := calculator.GetInt64Med(filteredCollectedPrices)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get median")
			return err
		}
		log.Debug().Str("Player", "Aggregator").Int64("roundId", priceDataMessage.RoundID).Int64("global_aggregate", median).Msg("global aggregated")
		err = n.insertGlobalAggregate(median, priceDataMessage.RoundID)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate")
			return err
		}
	}
	return nil
}

func (n *Aggregator) getLatestLocalAggregate(ctx context.Context) (int64, time.Time, error) {
	redisAggregate, err := GetLatestLocalAggregateFromRdb(ctx, n.Name)
	if err != nil {
		pgsqlAggregate, err := GetLatestLocalAggregateFromPgs(ctx, n.Name)
		if err != nil {
			return 0, time.Time{}, err
		}
		return pgsqlAggregate.Value, pgsqlAggregate.Timestamp, nil
	}
	return redisAggregate.Value, redisAggregate.Timestamp, nil
}

func (n *Aggregator) getLatestRoundId(ctx context.Context) (int64, error) {
	result, err := db.QueryRow[globalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"name": n.Name})
	if err != nil {
		return 0, err
	}
	return result.Round, nil
}

func (n *Aggregator) insertGlobalAggregate(value int64, round int64) error {
	err := n.insertPgsql(n.nodeCtx, value, round)
	if err != nil {
		return err
	}
	return n.insertRdb(n.nodeCtx, value, round)
}

func (n *Aggregator) insertPgsql(ctx context.Context, value int64, round int64) error {
	return db.QueryWithoutResult(n.nodeCtx, InsertGlobalAggregateQuery, map[string]any{"name": n.Name, "value": value, "round": round})
}

func (n *Aggregator) insertRdb(ctx context.Context, value int64, round int64) error {
	key := "globalAggregate:" + n.Name
	data, err := json.Marshal(redisGlobalAggregate{Value: value, Round: round})
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal global aggregate")
		return err
	}
	return db.Set(ctx, key, string(data), time.Duration(5*time.Minute))
}

func (n *Aggregator) executeDeviation() error {
	// signals for deviation job which triggers immediate aggregation and sends submission request to submitter
	return nil
}
