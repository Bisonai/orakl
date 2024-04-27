package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewAggregator(h host.Host, ps *pubsub.PubSub, topicString string, config AggregatorConfig) (*Aggregator, error) {
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

	aggregateInterval := time.Duration(config.AggregateInterval) * time.Millisecond

	aggregator := Aggregator{
		AggregatorConfig:        config,
		Raft:                    raft.NewRaftNode(h, ps, topic, 100, aggregateInterval),
		CollectedPrices:         map[int64][]int64{},
		CollectedProofs:         map[int64][][]byte{},
		CollectedAgreements:     map[int64][]bool{},
		PreparedLocalAggregates: map[int64]int64{},
		SyncedTimes:             map[int64]time.Time{},
		AggregatorMutex:         sync.Mutex{},
		RoundID:                 0,
		SignHelper:              signHelper,
	}
	aggregator.Raft.LeaderJob = aggregator.LeaderJob
	aggregator.Raft.HandleCustomMessage = aggregator.HandleCustomMessage

	return &aggregator, nil
}

func (n *Aggregator) Run(ctx context.Context) {
	latestRoundId, err := getLatestRoundId(ctx, n.ID)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get latest round id, setting roundId to 1")
		n.RoundID = 1
	} else if latestRoundId > 0 {
		n.RoundID = latestRoundId
	}

	n.Raft.Run(ctx)
}

func (n *Aggregator) LeaderJob() error {
	n.RoundID++
	n.Raft.IncreaseTerm()
	return n.PublishSyncMessage(n.RoundID, time.Now())
}

func (n *Aggregator) HandleCustomMessage(ctx context.Context, message raft.Message) error {
	switch message.Type {
	case RoundSync:
		return n.HandleRoundSyncMessage(ctx, message)
	case SyncReply:
		return n.HandleSyncReplyMessage(ctx, message)
	case Trigger:
		return n.HandleTriggerMessage(ctx, message)
	case PriceData:
		return n.HandlePriceDataMessage(ctx, message)
	case ProofMsg:
		return n.HandleProofMessage(ctx, message)
	default:
		return fmt.Errorf("unknown message type received in HandleCustomMessage: %v", message.Type)
	}
}

func (n *Aggregator) HandleRoundSyncMessage(ctx context.Context, msg raft.Message) error {
	var roundSyncMessage RoundSyncMessage
	err := json.Unmarshal(msg.Data, &roundSyncMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal round sync message")
		return err
	}

	if msg.SentFrom != n.Raft.GetLeader() {
		log.Warn().Str("Player", "Aggregator").Msg("round sync message sent from non-leader")
		return fmt.Errorf("round sync message sent from non-leader")
	}

	if roundSyncMessage.LeaderID == "" || roundSyncMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid round sync message")
		return fmt.Errorf("invalid round sync message: %v", roundSyncMessage)
	}

	if n.Raft.GetRole() != raft.Leader {
		n.RoundID = roundSyncMessage.RoundID
	}

	n.AggregatorMutex.Lock()
	defer n.AggregatorMutex.Unlock()

	value, updateTime, err := GetLatestLocalAggregate(ctx, n.ID)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get latest local aggregate")
		// set value to -1 rather than returning error
		// it is to proceed further steps even if current node fails to get latest local aggregate
		// if not enough messages collected from HandleSyncReplyMessage, it will hang in certain round
		value = -1
	}

	n.PreparedLocalAggregates[roundSyncMessage.RoundID] = value
	n.SyncedTimes[roundSyncMessage.RoundID] = roundSyncMessage.Timestamp

	if !n.isTimeValid(updateTime, roundSyncMessage.Timestamp) {
		n.PreparedLocalAggregates[roundSyncMessage.RoundID] = -1
		return n.PublishSyncReplyMessage(roundSyncMessage.RoundID, false)
	}
	return n.PublishSyncReplyMessage(roundSyncMessage.RoundID, true)
}

func (n *Aggregator) HandleSyncReplyMessage(ctx context.Context, msg raft.Message) error {
	if n.Raft.GetRole() != raft.Leader {
		log.Debug().Str("Player", "Aggregator").Msg("received sync reply message while not leader")
		return nil
	}

	var syncReplyMessage SyncReplyMessage
	err := json.Unmarshal(msg.Data, &syncReplyMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal sync reply message")
		return err
	}

	if syncReplyMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid sync reply message")
		return fmt.Errorf("invalid sync reply message: %v", syncReplyMessage)
	}

	n.AggregatorMutex.Lock()
	defer n.AggregatorMutex.Unlock()

	if _, ok := n.CollectedAgreements[syncReplyMessage.RoundID]; !ok {
		n.CollectedAgreements[syncReplyMessage.RoundID] = []bool{}
	}

	n.CollectedAgreements[syncReplyMessage.RoundID] = append(n.CollectedAgreements[syncReplyMessage.RoundID], syncReplyMessage.Agreed)
	if len(n.CollectedAgreements[syncReplyMessage.RoundID]) >= n.Raft.SubscribersCount()+1 {
		defer delete(n.CollectedAgreements, syncReplyMessage.RoundID)
		agreeCount := 0
		for _, agreed := range n.CollectedAgreements[syncReplyMessage.RoundID] {
			if agreed {
				agreeCount++
			}
		}
		requiredAgreements := int(float64(n.Raft.SubscribersCount()) * AGREEMENT_QUORUM)
		if agreeCount >= requiredAgreements {
			return n.PublishTriggerMessage(syncReplyMessage.RoundID)
		} else {
			log.Warn().Str("Player", "Aggregator").Int("agreeCount", agreeCount).Int("requiredAgreements", requiredAgreements).Msg("not enough agreements, resigning as leader")
			n.Raft.StopHeartbeatTicker()
			n.Raft.UpdateRole(raft.Follower)
			return nil
		}
	}
	return nil
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
		return fmt.Errorf("invalid trigger message: %v", triggerMessage)
	}

	if msg.SentFrom != n.Raft.GetLeader() {
		log.Warn().Str("Player", "Aggregator").Msg("trigger message sent from non-leader")
		return fmt.Errorf("trigger message sent from non-leader")
	}
	defer delete(n.PreparedLocalAggregates, triggerMessage.RoundID)
	return n.PublishPriceDataMessage(triggerMessage.RoundID, n.PreparedLocalAggregates[triggerMessage.RoundID])
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
		defer delete(n.SyncedTimes, priceDataMessage.RoundID)
		filteredCollectedPrices := FilterNegative(n.CollectedPrices[priceDataMessage.RoundID])

		median, err := calculator.GetInt64Med(filteredCollectedPrices)
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to get median")
			return err
		}
		log.Debug().Str("Player", "Aggregator").Int64("roundId", priceDataMessage.RoundID).Int64("global_aggregate", median).Msg("global aggregated")
		err = InsertGlobalAggregate(ctx, n.ID, median, priceDataMessage.RoundID, n.SyncedTimes[priceDataMessage.RoundID])
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert global aggregate")
			return err
		}

		proof, err := n.SignHelper.MakeGlobalAggregateProof(median, n.SyncedTimes[priceDataMessage.RoundID])
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to make global aggregate proof")
			return err
		}
		return n.PublishProofMessage(priceDataMessage.RoundID, proof)
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
		err := InsertProof(ctx, n.ID, proofMessage.RoundID, n.CollectedProofs[proofMessage.RoundID])
		if err != nil {
			log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to insert proof")
			return err
		}
	}
	return nil
}

func (n *Aggregator) PublishSyncMessage(roundId int64, timestamp time.Time) error {
	roundMessage := RoundSyncMessage{
		LeaderID:  n.Raft.GetHostId(),
		RoundID:   roundId,
		Timestamp: timestamp,
	}

	marshalledRoundMessage, err := json.Marshal(roundMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal round message")
		return err
	}

	message := raft.Message{
		Type:     RoundSync,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledRoundMessage),
	}

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) PublishSyncReplyMessage(roundId int64, agreed bool) error {
	syncReplyMessage := SyncReplyMessage{
		RoundID: roundId,
		Agreed:  agreed,
	}

	marshalledSyncReplyMessage, err := json.Marshal(syncReplyMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal sync reply message")
		return err
	}

	message := raft.Message{
		Type:     SyncReply,
		SentFrom: n.Raft.GetHostId(),
		Data:     json.RawMessage(marshalledSyncReplyMessage),
	}

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) PublishTriggerMessage(roundId int64) error {
	triggerMessage := TriggerMessage{
		LeaderID: n.Raft.GetHostId(),
		RoundID:  roundId,
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

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) PublishPriceDataMessage(roundId int64, value int64) error {
	priceDataMessage := PriceDataMessage{
		RoundID:   roundId,
		PriceData: value,
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

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) PublishProofMessage(roundId int64, proof []byte) error {
	proofMessage := ProofMessage{
		RoundID: roundId,
		Proof:   proof,
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

	return n.Raft.PublishMessage(message)
}

func (n *Aggregator) isTimeValid(timeToValidate time.Time, baseTime time.Time) bool {
	aggregatorInterval := time.Duration(n.AggregateInterval) * time.Millisecond
	return timeToValidate.After(baseTime.Add(-aggregatorInterval)) && timeToValidate.Before(baseTime)
}
