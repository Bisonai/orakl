package aggregator

import (
	"bytes"
	"context"
	"encoding/json"

	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewAggregator(h host.Host, ps *pubsub.PubSub, topicString string, config Config, signHelper *helper.Signer, latestLocalAggregates *LatestLocalAggregates) (*Aggregator, error) {
	if h == nil || ps == nil || topicString == "" {
		return nil, errorSentinel.ErrAggregatorInvalidInitValue
	}

	topic, err := ps.Join(topicString)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("Failed to join topic")
		return nil, err
	}

	aggregateInterval := time.Duration(config.AggregateInterval) * time.Millisecond

	aggregator := Aggregator{
		Config: config,
		Raft:   raft.NewRaftNode(h, ps, topic, 1000, aggregateInterval),

		roundPrices: &RoundPrices{prices: map[int32][]int64{}},
		roundProofs: &RoundProofs{proofs: map[int32][][]byte{}},

		RoundID:               1,
		Signer:                signHelper,
		LatestLocalAggregates: latestLocalAggregates,
	}
	aggregator.Raft.LeaderJob = aggregator.LeaderJob
	aggregator.Raft.HandleCustomMessage = aggregator.HandleCustomMessage

	return &aggregator, nil
}

func (n *Aggregator) Run(ctx context.Context) {
	latestRoundId, err := getLatestRoundId(ctx, n.ID)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get latest round id, setting roundId to 1")
	} else if latestRoundId > 0 {
		n.RoundID = latestRoundId
	}

	n.Raft.Run(ctx)
}

func (n *Aggregator) LeaderJob(ctx context.Context) error {
	n.RoundID++
	n.Raft.IncreaseTerm()
	return n.PublishTriggerMessage(ctx, n.RoundID, time.Now())
}

func (n *Aggregator) HandleCustomMessage(ctx context.Context, message raft.Message) error {
	switch message.Type {
	case Trigger:
		return n.HandleTriggerMessage(ctx, message)
	case PriceData:
		return n.HandlePriceDataMessage(ctx, message)
	case ProofMsg:
		return n.HandleProofMessage(ctx, message)
	default:
		return errorSentinel.ErrAggregatorUnhandledCustomMessage
	}
}

func (n *Aggregator) HandleTriggerMessage(ctx context.Context, msg raft.Message) error {
	var triggerMessage TriggerMessage
	err := json.Unmarshal(msg.Data, &triggerMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal trigger message")
		return err
	}

	if triggerMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid trigger message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	if msg.SentFrom != n.Raft.GetLeader() {
		log.Warn().Str("Player", "Aggregator").Msg("trigger message sent from non-leader")
		return errorSentinel.ErrAggregatorNonLeaderRaftMessage
	}

	var value int64
	localAggregate, ok := n.LatestLocalAggregates.Load(n.ID)
	if !ok {
		log.Error().Str("Player", "Aggregator").Msg("failed to get latest local aggregate")
		// set value to -1 rather than returning error
		// it is to proceed further steps even if current node fails to get latest local aggregate
		// if not enough messages collected from HandleSyncReplyMessage, it will hang in certain round
		value = -1
	} else {
		value = localAggregate.Value
	}

	return n.PublishPriceDataMessage(ctx, triggerMessage.RoundID, value, triggerMessage.Timestamp)
}

func (n *Aggregator) HandlePriceDataMessage(ctx context.Context, msg raft.Message) error {
	var priceDataMessage PriceDataMessage
	err := json.Unmarshal(msg.Data, &priceDataMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal price data message")
		return err
	}

	if priceDataMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid price data message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	n.roundPrices.mu.Lock()
	defer n.roundPrices.mu.Unlock()

	if prices, ok := n.roundPrices.prices[priceDataMessage.RoundID]; ok {
		n.roundPrices.prices[priceDataMessage.RoundID] = append(prices, priceDataMessage.PriceData)
	} else {
		n.roundPrices.prices[priceDataMessage.RoundID] = []int64{priceDataMessage.PriceData}
	}
	if len(n.roundPrices.prices[priceDataMessage.RoundID]) >= n.Raft.SubscribersCount()+1 {
		defer delete(n.roundPrices.prices, priceDataMessage.RoundID)
		prices := n.roundPrices.prices[priceDataMessage.RoundID]
		log.Debug().Str("Player", "Aggregator").Int("peerCount", n.Raft.SubscribersCount()).Str("Name", n.Name).Any("collected prices", prices).Int32("roundId", priceDataMessage.RoundID).Msg("collected prices")

		filteredCollectedPrices := FilterNegative(prices)
		if len(filteredCollectedPrices) == 0 {
			log.Warn().Str("Player", "Aggregator").Str("Name", n.Name).Int32("roundId", priceDataMessage.RoundID).Msg("no prices collected")
			return nil
		}

		median, err := calculator.GetInt64Med(filteredCollectedPrices)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get median")
			return err
		}
		log.Debug().Str("Player", "Aggregator").Str("Name", n.Name).Any("filtered collected prices", filteredCollectedPrices).Int32("roundId", priceDataMessage.RoundID).Int64("global_aggregate", median).Msg("global aggregated")

		proof, err := n.Signer.MakeGlobalAggregateProof(median, priceDataMessage.Timestamp, n.Name)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to make global aggregate proof")
			return err
		}
		return n.PublishProofMessage(ctx, priceDataMessage.RoundID, median, proof, priceDataMessage.Timestamp)
	}
	return nil
}

func (n *Aggregator) HandleProofMessage(ctx context.Context, msg raft.Message) error {
	var proofMessage ProofMessage
	err := json.Unmarshal(msg.Data, &proofMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal proof message")
		return err
	}

	if proofMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid proof message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	if proofMessage.Proof == nil {
		log.Error().Str("Player", "Aggregator").Msg("invalid proof message")
		return errorSentinel.ErrAggregatorEmptyProof
	}

	n.roundProofs.mu.Lock()
	defer n.roundProofs.mu.Unlock()

	if proofs, ok := n.roundProofs.proofs[proofMessage.RoundID]; ok {
		n.roundProofs.proofs[proofMessage.RoundID] = append(proofs, proofMessage.Proof)
	} else {
		n.roundProofs.proofs[proofMessage.RoundID] = [][]byte{proofMessage.Proof}
	}

	if len(n.roundProofs.proofs[proofMessage.RoundID]) >= n.Raft.SubscribersCount()+1 {
		defer delete(n.roundProofs.proofs, proofMessage.RoundID)
		log.Debug().Str("Player", "Aggregator").Str("Name", n.Name).Int("peerCount", n.Raft.SubscribersCount()).Int32("roundId", proofMessage.RoundID).Any("collected proofs", n.roundProofs.proofs[proofMessage.RoundID]).Msg("collected proofs")

		globalAggregate := GlobalAggregate{
			ConfigID:  n.ID,
			Value:     proofMessage.Value,
			Round:     proofMessage.RoundID,
			Timestamp: proofMessage.Timestamp}

		concatProof := bytes.Join(n.roundProofs.proofs[proofMessage.RoundID], nil)
		proof := Proof{ConfigID: n.ID, Round: proofMessage.RoundID, Proof: concatProof}

		go func(ctx context.Context, globalAggregate GlobalAggregate, proof Proof) {
			err := PublishGlobalAggregateAndProof(ctx, globalAggregate, proof)
			if err != nil {
				log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to publish global aggregate and proof")
			}
		}(ctx, globalAggregate, proof)

	}
	return nil
}

func (n *Aggregator) PublishTriggerMessage(ctx context.Context, roundId int32, timestamp time.Time) error {
	triggerMessage := TriggerMessage{
		LeaderID:  n.Raft.GetHostId(),
		RoundID:   roundId,
		Timestamp: timestamp,
	}

	marshalledTriggerMessage, err := json.Marshal(triggerMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal trigger message")
		return err
	}

	message := raft.Message{
		Type:     Trigger,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledTriggerMessage),
	}

	return n.Raft.PublishMessage(ctx, message)
}

func (n *Aggregator) PublishPriceDataMessage(ctx context.Context, roundId int32, value int64, timestamp time.Time) error {
	priceDataMessage := PriceDataMessage{
		RoundID:   roundId,
		PriceData: value,
		Timestamp: timestamp,
	}

	marshalledPriceDataMessage, err := json.Marshal(priceDataMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal price data message")
		return err
	}

	message := raft.Message{
		Type:     PriceData,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledPriceDataMessage),
	}

	return n.Raft.PublishMessage(ctx, message)
}

func (n *Aggregator) PublishProofMessage(ctx context.Context, roundId int32, value int64, proof []byte, timestamp time.Time) error {
	proofMessage := ProofMessage{
		RoundID:   roundId,
		Value:     value,
		Proof:     proof,
		Timestamp: timestamp,
	}

	marshalledProofMessage, err := json.Marshal(proofMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal proof message")
		return err
	}

	message := raft.Message{
		Type:     ProofMsg,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledProofMessage),
	}

	return n.Raft.PublishMessage(ctx, message)
}
