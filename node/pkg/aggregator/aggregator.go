package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const LEADER_TIMEOUT = 5 * time.Second

func NewAggregator(h host.Host, ps *pubsub.PubSub, topicString string) (*Aggregator, error) {
	if h == nil || ps == nil || topicString == "" {
		return nil, fmt.Errorf("invalid parameters")
	}

	topic, err := ps.Join(topicString)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("Failed to join topic")
		return nil, err
	}

	signHelper, err := helper.NewSignHelper("")
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to create sign helper")
		return nil, err
	}

	aggregator := Aggregator{
		Raft:            raft.NewRaftNode(h, ps, topic, 100, LEADER_TIMEOUT),
		CollectedPrices: map[int64][]int64{},
		CollectedProofs: map[int64][][]byte{},
		AggregatorMutex: sync.Mutex{},
		RoundID:         0,
		SignHelper:      signHelper,
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
	return n.PublishRoundMessage(n.RoundID)
}

func (n *Aggregator) HandleCustomMessage(message raft.Message) error {
	switch message.Type {
	case RoundSync:
		return n.HandleRoundSyncMessage(message)
	case PriceData:
		return n.HandlePriceDataMessage(message)
	case ProofMsg:
		return n.HandleProofMessage(message)
	default:
		return fmt.Errorf("unknown message type received in HandleCustomMessage: %v", message.Type)
	}
}

func (n *Aggregator) HandleRoundSyncMessage(msg raft.Message) error {
	var roundSyncMessage RoundSyncMessage
	err := json.Unmarshal(msg.Data, &roundSyncMessage)
	if err != nil {
		return err
	}

	if roundSyncMessage.LeaderID == "" || roundSyncMessage.RoundID == 0 {
		return fmt.Errorf("invalid round sync message: %v", roundSyncMessage)
	}

	if n.Raft.GetRole() != raft.Leader {
		n.RoundID = roundSyncMessage.RoundID
	}
	var updateValue int64 = -1
	value, updateTime, err := GetLatestLocalAggregate(n.nodeCtx, n.Name)

	if err == nil && n.LastLocalAggregateTime.IsZero() || !n.LastLocalAggregateTime.Equal(updateTime) {
		updateValue = value
		n.LastLocalAggregateTime = updateTime
	}

	return n.PublishPriceDataMessage(n.RoundID, updateValue)
}

func (n *Aggregator) HandlePriceDataMessage(msg raft.Message) error {
	var priceDataMessage PriceDataMessage
	err := json.Unmarshal(msg.Data, &priceDataMessage)
	if err != nil {
		return err
	}

	if priceDataMessage.RoundID == 0 {
		return fmt.Errorf("invalid price data message: %v", priceDataMessage)
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
		err = InsertGlobalAggregate(n.nodeCtx, n.Name, median, priceDataMessage.RoundID)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate")
			return err
		}

		proof, err := n.SignHelper.MakeGlobalAggregateProof(median)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to make global aggregate proof")
			return err
		}
		return n.PublishProofMessage(priceDataMessage.RoundID, proof)
	}
	return nil
}

func (n *Aggregator) HandleProofMessage(msg raft.Message) error {
	var proofMessage ProofMessage
	err := json.Unmarshal(msg.Data, &proofMessage)
	if err != nil {
		return err
	}

	if proofMessage.RoundID == 0 {
		return fmt.Errorf("invalid proof message: %v", proofMessage)
	}

	n.AggregatorMutex.Lock()
	defer n.AggregatorMutex.Unlock()
	if _, ok := n.CollectedProofs[proofMessage.RoundID]; !ok {
		n.CollectedProofs[proofMessage.RoundID] = [][]byte{}
	}

	n.CollectedProofs[proofMessage.RoundID] = append(n.CollectedProofs[proofMessage.RoundID], proofMessage.Proof)
	if len(n.CollectedProofs[proofMessage.RoundID]) >= n.Raft.SubscribersCount()+1 {
		defer delete(n.CollectedProofs, proofMessage.RoundID)
		err := InsertProof(n.nodeCtx, n.Name, proofMessage.RoundID, n.CollectedProofs[proofMessage.RoundID])
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proof")
			return err
		}
	}
	return nil
}

func (n *Aggregator) getLatestRoundId(ctx context.Context) (int64, error) {
	result, err := db.QueryRow[globalAggregate](ctx, SelectLatestGlobalAggregateQuery, map[string]any{"name": n.Name})
	if err != nil {
		return 0, err
	}
	return result.Round, nil
}

func (n *Aggregator) PublishRoundMessage(roundId int64) error {
	roundMessage := RoundSyncMessage{
		LeaderID: n.Raft.GetHostId(),
		RoundID:  roundId,
	}

	marshalledRoundMessage, err := json.Marshal(roundMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     RoundSync,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledRoundMessage),
	}

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) PublishPriceDataMessage(roundId int64, value int64) error {
	priceDataMessage := PriceDataMessage{
		RoundID:   roundId,
		PriceData: value,
	}

	marshalledPriceDataMessage, err := json.Marshal(priceDataMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     PriceData,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledPriceDataMessage),
	}

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) PublishProofMessage(roundId int64, proof []byte) error {
	proofMessage := ProofMessage{
		RoundID: roundId,
		Proof:   proof,
	}

	marshalledProofMessage, err := json.Marshal(proofMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     ProofMsg,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledProofMessage),
	}

	return n.Raft.PublishMessage(message)
}
