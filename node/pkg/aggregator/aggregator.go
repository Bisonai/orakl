package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"time"

	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/chain/helper"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/raft"
	"bisonai.com/miko/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewAggregator(
	h host.Host,
	ps *pubsub.PubSub,
	topicString string,
	config Config,
	signHelper *helper.Signer,
	latestLocalAggregates *LatestLocalAggregates,
	mb *bus.MessageBus,
) (*Aggregator, error) {
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

		roundTriggers: &RoundTriggers{
			locked: map[int32]bool{},
		},
		roundPrices: &RoundPrices{
			prices:  map[int32][]int64{},
			senders: map[int32][]string{},
			locked:  map[int32]bool{},
		},
		roundPriceFixes: &RoundPriceFixes{
			locked: map[int32]bool{},
		},
		roundProofs: &RoundProofs{
			proofs:  map[int32][][]byte{},
			senders: map[int32][]string{},
			locked:  map[int32]bool{},
		},

		RoundID:               1,
		Signer:                signHelper,
		LatestLocalAggregates: latestLocalAggregates,
		bus:                   mb,
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
		n.RoundID = latestRoundId + 1
	}

	n.Raft.Run(ctx)
}

func (n *Aggregator) LeaderJob(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RoundID += 1

	return n.PublishTriggerMessage(ctx, n.RoundID, time.Now())
}

func (n *Aggregator) HandleCustomMessage(ctx context.Context, message raft.Message) error {
	switch message.Type {
	case Trigger:
		return n.HandleTriggerMessage(ctx, message)
	case PriceData:
		return n.HandlePriceDataMessage(ctx, message)
	case PriceFix:
		return n.HandlePriceFixMessage(ctx, message)
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
	defer n.leaveOnlyLast10Entries(triggerMessage.RoundID)
	n.mu.Lock()
	defer n.mu.Unlock()
	if msg.SentFrom != n.Raft.GetHostId() {
		// follower can be changed into leader unexpectedly before recieving the message
		// increase round id before checking the message sent from leader
		// so that the next round will be triggered with larger round id
		// prevents already handled message error
		n.RoundID = max(triggerMessage.RoundID, n.RoundID)
	}

	if triggerMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid trigger message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	if msg.SentFrom != n.Raft.GetLeader() {
		log.Warn().Str("Player", "Aggregator").Msg("trigger message sent from non-leader")
		return errorSentinel.ErrAggregatorNonLeaderRaftMessage
	}

	n.roundTriggers.mu.Lock()
	defer n.roundTriggers.mu.Unlock()

	if n.roundTriggers.locked[triggerMessage.RoundID] {
		log.Warn().Str("Player", "Aggregator").Int32("RoundID", triggerMessage.RoundID).Msg("trigger message already processed")
		return nil
	}
	n.roundTriggers.locked[triggerMessage.RoundID] = true

	var value int64
	localAggregate, ok := n.LatestLocalAggregates.Load(n.ID)
	if !ok {
		log.Warn().Str("Player", "Aggregator").Msg("failed to get latest local aggregate")
		// set value to -1 rather than returning error
		// it is to proceed further steps even if current node fails to get latest local aggregate
		// if not enough messages collected from HandleSyncReplyMessage, it will hang in certain round
		value = -1
	} else {
		n.roundLocalAggregate[triggerMessage.RoundID] = localAggregate.Value
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

	if n.roundPrices.locked[priceDataMessage.RoundID] || n.roundPrices.isReplay(priceDataMessage.RoundID, msg.SentFrom) {
		log.Warn().Str("Player", "Aggregator").Int32("RoundID", priceDataMessage.RoundID).Msg("price data message already processed")
		return nil
	}

	if prices, ok := n.roundPrices.prices[priceDataMessage.RoundID]; ok {
		n.roundPrices.prices[priceDataMessage.RoundID] = append(prices, priceDataMessage.PriceData)
		n.roundPrices.senders[priceDataMessage.RoundID] = append(n.roundPrices.senders[priceDataMessage.RoundID], msg.SentFrom)
	} else {
		n.roundPrices.prices[priceDataMessage.RoundID] = []int64{priceDataMessage.PriceData}
		n.roundPrices.senders[priceDataMessage.RoundID] = []string{msg.SentFrom}
	}

	if len(n.roundPrices.prices[priceDataMessage.RoundID]) == n.Raft.SubscribersCount()+1 {
		n.roundPrices.locked[priceDataMessage.RoundID] = true

		if n.Raft.GetRole() == raft.Leader {
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

			return n.PublishPriceFixMessage(ctx, priceDataMessage.RoundID, median, priceDataMessage.Timestamp)
		}
	}
	return nil
}

func (n *Aggregator) HandlePriceFixMessage(ctx context.Context, msg raft.Message) error {
	var priceFixMessage PriceFixMessage
	err := json.Unmarshal(msg.Data, &priceFixMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal price fix message")
		return err
	}

	if msg.SentFrom != n.Raft.GetLeader() {
		log.Warn().Str("Player", "Aggregator").Str("Leader", n.Raft.GetLeader()).Msg("price fix message sent from non-leader")
		return errorSentinel.ErrAggregatorNonLeaderRaftMessage
	}

	n.roundPriceFixes.mu.Lock()
	defer n.roundPriceFixes.mu.Unlock()
	if n.roundPriceFixes.locked[priceFixMessage.RoundID] {
		log.Warn().Str("Player", "Aggregator").Int32("RoundID", priceFixMessage.RoundID).Msg("price fix message already processed")
		return nil
	}

	n.roundPriceFixes.locked[priceFixMessage.RoundID] = true

	proof, err := n.Signer.MakeGlobalAggregateProof(priceFixMessage.PriceData, priceFixMessage.Timestamp, n.Name)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to make global aggregate proof")
		return err
	}

	return n.PublishProofMessage(ctx, priceFixMessage.RoundID, priceFixMessage.PriceData, proof, priceFixMessage.Timestamp)

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

	if n.roundProofs.locked[proofMessage.RoundID] || n.roundProofs.isReplay(proofMessage.RoundID, msg.SentFrom) {
		log.Warn().Str("Player", "Aggregator").Int32("RoundID", proofMessage.RoundID).Msg("proof message already processed")
		return nil
	}

	if proofs, ok := n.roundProofs.proofs[proofMessage.RoundID]; ok {
		n.roundProofs.proofs[proofMessage.RoundID] = append(proofs, proofMessage.Proof)
		n.roundProofs.senders[proofMessage.RoundID] = append(n.roundProofs.senders[proofMessage.RoundID], msg.SentFrom)
	} else {
		n.roundProofs.proofs[proofMessage.RoundID] = [][]byte{proofMessage.Proof}
		n.roundProofs.senders[proofMessage.RoundID] = []string{msg.SentFrom}
	}

	if len(n.roundProofs.proofs[proofMessage.RoundID]) == n.Raft.SubscribersCount()+1 {
		n.roundProofs.locked[proofMessage.RoundID] = true

		log.Debug().Str("Player", "Aggregator").Str("Name", n.Name).Int("peerCount", n.Raft.SubscribersCount()).Int32("roundId", proofMessage.RoundID).Any("collected proofs", n.roundProofs.proofs[proofMessage.RoundID]).Msg("collected proofs")

		globalAggregate := GlobalAggregate{
			ConfigID:  n.ID,
			Value:     proofMessage.Value,
			Round:     proofMessage.RoundID,
			Timestamp: proofMessage.Timestamp}

		concatProof := bytes.Join(n.roundProofs.proofs[proofMessage.RoundID], nil)
		proof := Proof{ConfigID: n.ID, Round: proofMessage.RoundID, Proof: concatProof}

		n.mu.Lock()
		defer n.mu.Unlock()
		if math.Abs(float64(n.roundLocalAggregate[proofMessage.RoundID]-globalAggregate.Value))/float64(globalAggregate.Value) > 0.3 {
			log.Warn().Str("Player", "Aggregator").Str("Name", n.Name).Int32("roundId", proofMessage.RoundID).Int64("localAggregate", n.roundLocalAggregate[proofMessage.RoundID]).Int64("globalAggregate", globalAggregate.Value).Msg("local aggregate and global aggregate mismatch")
			msg := bus.Message{
				From: bus.AGGREGATOR,
				To:   bus.FETCHER,
				Content: bus.MessageContent{
					Command: bus.REFRESH_FETCHER_APP,
					Args:    nil,
				},
			}
			err = n.bus.Publish(msg)
			if err != nil {
				log.Warn().Str("Player", "Aggregator").Err(err).Msg("failed to publish fetcher refresh bus message")
			}
		}

		err := PublishGlobalAggregateAndProof(ctx, n.Name, globalAggregate, proof)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to publish global aggregate and proof")
		}
		delete(n.roundLocalAggregate, proofMessage.RoundID)
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

func (n *Aggregator) PublishPriceFixMessage(ctx context.Context, roundId int32, value int64, timestamp time.Time) error {
	priceFixMessage := PriceFixMessage{
		RoundID:   roundId,
		PriceData: value,
		Timestamp: timestamp,
	}

	marshalledPriceFixMessage, err := json.Marshal(priceFixMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal price fix message")
		return err
	}

	message := raft.Message{
		Type:     PriceFix,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledPriceFixMessage),
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

func (n *Aggregator) leaveOnlyLast10Entries(roundID int32) {
	n.roundTriggers.leaveOnlyLast10Entries(roundID)
	n.roundPrices.leaveOnlyLast10Entries(roundID)
	n.roundPriceFixes.leaveOnlyLast10Entries(roundID)
	n.roundProofs.leaveOnlyLast10Entries(roundID)
}
