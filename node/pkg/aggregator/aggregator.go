package aggregator

import (
	"bytes"
	"context"
	"encoding/json"

	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/raft"
	"bisonai.com/miko/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const maxLeaderMsgReceiveTimeout = 100 * time.Millisecond

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

	if triggerMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid trigger message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	currentLeader := n.Raft.GetLeader()
	if msg.SentFrom != currentLeader {
		log.Warn().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("CurrentLeader", currentLeader).Str("Me", n.Raft.GetHostId()).Msg("trigger message sent from non-leader")
		return errorSentinel.ErrAggregatorNonLeaderRaftMessage
	}

	if msg.SentFrom != n.Raft.GetHostId() {
		n.mu.Lock()
		n.RoundID = max(triggerMessage.RoundID, n.RoundID)
		n.mu.Unlock()
	}

	n.roundTriggers.mu.Lock()
	defer n.roundTriggers.mu.Unlock()

	if n.roundTriggers.locked[triggerMessage.RoundID] {
		log.Warn().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("CurrentLeader", currentLeader).Str("Me", n.Raft.GetHostId()).Int32("RoundID", triggerMessage.RoundID).Msg("trigger message already processed")
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

	n.storeRoundPriceData(priceDataMessage.RoundID, priceDataMessage.PriceData, msg.SentFrom)

	if len(n.roundPrices.prices[priceDataMessage.RoundID]) == n.Raft.SubscribersCount()+1 {
		// if all messsages received for the round
		return n.processCollectedPrices(ctx, priceDataMessage.RoundID, priceDataMessage.Timestamp)
	} else if len(n.roundPrices.prices[priceDataMessage.RoundID]) == 1 {
		// if it's first message for the round
		go n.startPriceCollectionTimeout(ctx, priceDataMessage.RoundID, priceDataMessage.Timestamp)
	}

	return nil
}

func (n *Aggregator) storeRoundPriceData(roundID int32, priceData int64, sender string) {
	if prices, ok := n.roundPrices.prices[roundID]; ok {
		n.roundPrices.prices[roundID] = append(prices, priceData)
		n.roundPrices.senders[roundID] = append(n.roundPrices.senders[roundID], sender)
	} else {
		n.roundPrices.prices[roundID] = []int64{priceData}
		n.roundPrices.senders[roundID] = []string{sender}
	}
}

func (n *Aggregator) startPriceCollectionTimeout(ctx context.Context, roundID int32, timestamp time.Time) {
	timer := time.NewTimer(maxLeaderMsgReceiveTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		n.roundPrices.mu.Lock()
		defer n.roundPrices.mu.Unlock()

		if !n.roundPrices.locked[roundID] && len(n.roundPrices.prices[roundID]) >= (n.Raft.SubscribersCount()+1)/2 {
			log.Debug().Str("Player", "Aggregator").Int32("roundId", roundID).Msg("timeout reached, processing available prices")
			err := n.processCollectedPrices(ctx, roundID, timestamp)
			if err != nil {
				log.Error().Err(err).Int32("roundId", roundID).Msg("failed to process collected prices")
			}
		}
	case <-ctx.Done():
		return
	}
}

func (n *Aggregator) processCollectedPrices(ctx context.Context, roundID int32, timestamp time.Time) error {
	n.roundPrices.locked[roundID] = true
	if n.Raft.GetRole() != raft.Leader {
		return nil
	}

	prices := n.roundPrices.prices[roundID]
	log.Debug().Str("Player", "Aggregator").Int("peerCount", n.Raft.SubscribersCount()).Str("Name", n.Name).Any("collected prices", prices).Int32("roundId", roundID).Msg("collected prices")

	filteredCollectedPrices := FilterNegative(prices)
	if len(filteredCollectedPrices) == 0 {
		log.Warn().Str("Player", "Aggregator").Str("Name", n.Name).Int32("roundId", roundID).Msg("no prices collected")
		return nil
	}

	median, err := calculator.GetInt64Med(filteredCollectedPrices)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get median")
		return err
	}

	return n.PublishPriceFixMessage(ctx, roundID, median, timestamp)
}

func (n *Aggregator) HandlePriceFixMessage(ctx context.Context, msg raft.Message) error {
	var priceFixMessage PriceFixMessage
	err := json.Unmarshal(msg.Data, &priceFixMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal price fix message")
		return err
	}

	currentLeader := n.Raft.GetLeader()
	if msg.SentFrom != currentLeader {
		log.Warn().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("CurrentLeader", currentLeader).Str("Me", n.Raft.GetHostId()).Msg("price fix message sent from non-leader")
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

	n.storeRoundProofData(proofMessage.RoundID, proofMessage.Proof, msg.SentFrom)

	if len(n.roundProofs.proofs[proofMessage.RoundID]) == n.Raft.SubscribersCount()+1 {
		return n.processCollectedProofs(ctx, proofMessage)
	} else if len(n.roundProofs.proofs[proofMessage.RoundID]) == 1 {
		go n.startProofCollectionTimeout(ctx, proofMessage)
	}

	return nil
}

func (n *Aggregator) storeRoundProofData(roundID int32, proofData []byte, sender string) {
	if proofs, ok := n.roundProofs.proofs[roundID]; ok {
		n.roundProofs.proofs[roundID] = append(proofs, proofData)
		n.roundProofs.senders[roundID] = append(n.roundProofs.senders[roundID], sender)
	} else {
		n.roundProofs.proofs[roundID] = [][]byte{proofData}
		n.roundProofs.senders[roundID] = []string{sender}
	}
}

func (n *Aggregator) startProofCollectionTimeout(ctx context.Context, proofMessage ProofMessage) {
	timer := time.NewTimer(maxLeaderMsgReceiveTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		n.roundProofs.mu.Lock()
		defer n.roundProofs.mu.Unlock()

		if !n.roundProofs.locked[proofMessage.RoundID] && len(n.roundProofs.proofs[proofMessage.RoundID]) >= (n.Raft.SubscribersCount()+1)/2 {
			log.Debug().Str("Player", "Aggregator").Int32("roundId", proofMessage.RoundID).Msg("timeout reached, processing available proofs")
			err := n.processCollectedProofs(ctx, proofMessage)
			if err != nil {
				log.Error().Err(err).Int32("roundId", proofMessage.RoundID).Msg("failed to process collected proofs")
			}
		}
	case <-ctx.Done():
		log.Debug().Str("Player", "Aggregator").Int32("roundId", proofMessage.RoundID).Msg("context canceled, stopping timeout")
		return
	}
}

func (n *Aggregator) processCollectedProofs(ctx context.Context, proofMessage ProofMessage) error {
	n.roundProofs.locked[proofMessage.RoundID] = true
	log.Debug().Str("Player", "Aggregator").Str("Name", n.Name).Int("peerCount", n.Raft.SubscribersCount()).Int32("roundId", proofMessage.RoundID).Any("collected proofs", n.roundProofs.proofs[proofMessage.RoundID]).Msg("collected proofs")

	globalAggregate := GlobalAggregate{
		ConfigID:  n.ID,
		Value:     proofMessage.Value,
		Round:     proofMessage.RoundID,
		Timestamp: proofMessage.Timestamp}

	concatProof := bytes.Join(n.roundProofs.proofs[proofMessage.RoundID], nil)
	proof := Proof{ConfigID: n.ID, Round: proofMessage.RoundID, Proof: concatProof}

	err := PublishGlobalAggregateAndProof(ctx, n.Name, globalAggregate, proof)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to publish global aggregate and proof")
		return err
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
