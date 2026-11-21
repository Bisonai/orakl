package aggregator

import (
	"bytes"
	"context"

	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/metrics"
	"bisonai.com/miko/node/pkg/raft"
	"bisonai.com/miko/node/pkg/utils/calculator"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const maxLeaderMsgReceiveTimeout = 150 * time.Millisecond

// joinTopicAttempts/joinTopicRetryDelay bound retries of joining a topic that
// a just-stopped aggregator (refresh) is still closing asynchronously.
const (
	joinTopicAttempts   = 40
	joinTopicRetryDelay = 50 * time.Millisecond
)

// NewAggregator builds an aggregator. rounds persists RoundID advances; nil
// disables persistence (tests).
func NewAggregator(h host.Host, ps *pubsub.PubSub, topicString string, config Config, signHelper *helper.Signer, latestLocalAggregates *LatestLocalAggregates, rounds *roundPersister) (*Aggregator, error) {
	if h == nil || ps == nil || topicString == "" {
		return nil, errorSentinel.ErrAggregatorInvalidInitValue
	}

	topic, err := ps.Join(topicString)
	for attempt := 1; err != nil && attempt < joinTopicAttempts; attempt++ {
		time.Sleep(joinTopicRetryDelay)
		topic, err = ps.Join(topicString)
	}
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("Failed to join topic")
		return nil, err
	}

	aggregateInterval := time.Duration(config.AggregateInterval) * time.Millisecond

	aggregator := Aggregator{
		Config: config,
		Raft:   raft.NewRaftNode(h, ps, topic, 1000, aggregateInterval),

		roundTriggers: &RoundTriggers{
			locked: map[int32]roundTrigger{},
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
		rounds:                rounds,
	}
	aggregator.Raft.LeaderJob = aggregator.LeaderJob
	aggregator.Raft.HandleCustomMessage = aggregator.HandleCustomMessage
	// only the leader sends these; followers send price data and proofs
	aggregator.Raft.LeaderMessageTypes = map[raft.MessageType]bool{Trigger: true, PriceFix: true}

	return &aggregator, nil
}

func (n *Aggregator) Run(ctx context.Context) {
	if err := n.loadLatestRound(ctx); err != nil {
		return // ctx done
	}
	n.Raft.Run(ctx)
}

// loadLatestRoundRetryDelay is the wait between attempts to read the stored round.
var loadLatestRoundRetryDelay = time.Second

// loadLatestRound continues after the latest persisted (or, after a refresh,
// in-memory) round, so a restarted node never reuses a round it already used.
// A read error is retried until ctx is done: starting from round 1 instead
// would reissue used round IDs.
func (n *Aggregator) loadLatestRound(ctx context.Context) error {
	for {
		latestRoundId, err := n.rounds.Latest(ctx, n.Name)
		if err == nil {
			n.mu.Lock()
			// never go back: a restart of a running aggregator keeps its in-memory round
			n.RoundID = max(n.RoundID, latestRoundId+1)
			n.mu.Unlock()
			return nil
		}
		log.Error().Str("Player", "Aggregator").Str("name", n.Name).Err(err).Msg("failed to load latest round id, retrying")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(loadLatestRoundRetryDelay):
		}
	}
}

func (n *Aggregator) LeaderJob(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.RoundID += 1
	n.rounds.Advance(n.Name, n.RoundID)

	return n.PublishTriggerMessage(ctx, n.RoundID, n.now())
}

func (n *Aggregator) now() time.Time {
	if n.Now != nil {
		return n.Now()
	}
	return time.Now()
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
	err := decodeInner(msg.Data, &triggerMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal trigger message")
		return err
	}
	if triggerMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid trigger message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	currentLeader := n.Raft.GetLeader()
	if msg.SentFrom != currentLeader {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("CurrentLeader", currentLeader).Str("Me", n.Raft.GetHostId()).Msg("trigger message sent from non-leader")
		return errorSentinel.ErrAggregatorNonLeaderRaftMessage
	}

	// the leader's own trigger is its current round
	currentRoundID := triggerMessage.RoundID
	if msg.SentFrom != n.Raft.GetHostId() {
		n.mu.Lock()
		n.RoundID = max(triggerMessage.RoundID, n.RoundID)
		n.rounds.Advance(n.Name, n.RoundID)
		currentRoundID = n.RoundID
		n.mu.Unlock()
	}
	// prune only for accepted triggers: unvalidated input must not drop round state (#2853)
	defer n.pruneRounds(triggerMessage.RoundID, currentRoundID)

	n.roundTriggers.mu.Lock()
	defer n.roundTriggers.mu.Unlock()

	if prev, ok := n.roundTriggers.locked[triggerMessage.RoundID]; ok {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("CurrentLeader", currentLeader).Str("Me", n.Raft.GetHostId()).Int32("RoundID", triggerMessage.RoundID).Msg("trigger message already processed")
		if triggerMessage.RoundID < currentRoundID {
			// the leader is behind this node: report this node's round (no
			// price) so it catches up instead of retrying used rounds (#2867)
			return n.PublishPriceDataMessage(ctx, triggerMessage.RoundID, -1, triggerMessage.Timestamp, currentRoundID)
		}
		if msg.SentFrom != n.Raft.GetHostId() && !prev.same(msg.SentFrom, triggerMessage.Timestamp) {
			// another trigger for this node's current round: the leader (fresh
			// or lost round state) reuses a used round ID. Move past it and
			// report that, so the leader catches up past it too (#2927); a
			// duplicate delivery of the same trigger stays unanswered.
			n.mu.Lock()
			n.RoundID = max(n.RoundID, triggerMessage.RoundID+1)
			n.rounds.Advance(n.Name, n.RoundID)
			latestRoundID := n.RoundID
			n.mu.Unlock()
			log.Warn().Str("Player", "Aggregator").Str("Name", n.Name).Str("Sender", msg.SentFrom).Int32("RoundID", triggerMessage.RoundID).Int32("LatestRoundID", latestRoundID).Msg("leader reuses a used round ID")
			return n.PublishPriceDataMessage(ctx, triggerMessage.RoundID, -1, triggerMessage.Timestamp, latestRoundID)
		}
		return nil
	}
	n.roundTriggers.locked[triggerMessage.RoundID] = roundTrigger{sender: msg.SentFrom, timestamp: triggerMessage.Timestamp}

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

	return n.PublishPriceDataMessage(ctx, triggerMessage.RoundID, value, triggerMessage.Timestamp, currentRoundID)
}

func (n *Aggregator) HandlePriceDataMessage(ctx context.Context, msg raft.Message) error {
	var priceDataMessage PriceDataMessage
	err := decodeInner(msg.Data, &priceDataMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal price data message")
		return err
	}

	if priceDataMessage.RoundID == 0 {
		log.Error().Str("Player", "Aggregator").Msg("invalid price data message")
		return errorSentinel.ErrAggregatorInvalidRaftMessage
	}

	// before roundPrices.mu: LeaderJob holds n.mu while it publishes
	staleThrough := n.catchUpToPeer(msg.SentFrom, priceDataMessage.LatestRoundID)

	n.roundPrices.mu.Lock()
	defer n.roundPrices.mu.Unlock()

	if n.roundPrices.locked[priceDataMessage.RoundID] {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("Me", n.Raft.GetHostId()).Str("transmissionDelay", time.Since(msg.Timestamp).String()).Int32("RoundID", priceDataMessage.RoundID).Msg("price data message already processed")
		return nil
	}

	if priceDataMessage.RoundID <= staleThrough {
		// a peer already used this round ID: no new aggregate for it
		n.roundPrices.locked[priceDataMessage.RoundID] = true
		log.Warn().Str("Player", "Aggregator").Str("Name", n.Name).Str("Sender", msg.SentFrom).Int32("RoundID", priceDataMessage.RoundID).Int32("PeerRoundID", staleThrough).Msg("dropping round already used by a peer")
		return nil
	}

	if n.roundPrices.isReplay(priceDataMessage.RoundID, msg.SentFrom) {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("Me", n.Raft.GetHostId()).Int32("RoundID", priceDataMessage.RoundID).Msg("price data message replayed")
		return nil
	}

	n.storeRoundPriceData(priceDataMessage.RoundID, priceDataMessage.PriceData, msg.SentFrom)

	if len(n.roundPrices.prices[priceDataMessage.RoundID]) == n.Raft.SubscribersCount()+1 {
		// if all messsages received for the round
		return n.processCollectedPrices(ctx, priceDataMessage.RoundID, priceDataMessage.Timestamp)
	}

	if len(n.roundPrices.prices[priceDataMessage.RoundID]) == 1 {
		// if it's first message for the round
		go n.startPriceCollectionTimeout(ctx, priceDataMessage.RoundID, priceDataMessage.Timestamp)
	}

	return nil
}

// catchUpToPeer moves a lagging leader (fresh DB, or missed triggers as a
// follower) past peerRoundID, the round a peer reported using, so its next
// trigger is a round no peer has used (#2867). Followers keep their locks:
// re-running a used round ID with new prices would break the rule that a
// round ID is never reused (see loadLatestRound). It returns the highest round
// a peer was ahead at; this leader's rounds up to it are stale (0: none),
// whoever sent the message (also its own or an older node's price).
//
// Like a follower adopting an accepted trigger's RoundID, the jump has no
// upper bound (#2711).
func (n *Aggregator) catchUpToPeer(sender string, peerRoundID int32) int32 {
	if n.Raft.GetRole() != raft.Leader {
		return 0
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if sender != n.Raft.GetHostId() && peerRoundID > n.RoundID {
		log.Warn().Str("Player", "Aggregator").Str("Name", n.Name).Str("Sender", sender).Int32("RoundID", n.RoundID).Int32("PeerRoundID", peerRoundID).Msg("leader behind a peer, catching up")
		n.RoundID = peerRoundID
		n.rounds.Advance(n.Name, n.RoundID)
		n.peerRoundID = peerRoundID
	}
	return n.peerRoundID
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
		// a catch-up since the round started made it stale (#2867)
		n.mu.RLock()
		stale := roundID <= n.peerRoundID
		n.mu.RUnlock()

		n.roundPrices.mu.Lock()
		defer n.roundPrices.mu.Unlock()

		if stale {
			n.roundPrices.locked[roundID] = true
			return
		}
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
	err := decodeInner(msg.Data, &priceFixMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to unmarshal price fix message")
		return err
	}

	currentLeader := n.Raft.GetLeader()
	if msg.SentFrom != currentLeader {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("CurrentLeader", currentLeader).Str("Me", n.Raft.GetHostId()).Msg("price fix message sent from non-leader")
		return errorSentinel.ErrAggregatorNonLeaderRaftMessage
	}

	n.roundPriceFixes.mu.Lock()
	defer n.roundPriceFixes.mu.Unlock()
	if n.roundPriceFixes.locked[priceFixMessage.RoundID] {
		log.Debug().Str("Player", "Aggregator").Int32("RoundID", priceFixMessage.RoundID).Msg("price fix message already processed")
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
	err := decodeInner(msg.Data, &proofMessage)
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

	if n.roundProofs.locked[proofMessage.RoundID] {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("Me", n.Raft.GetHostId()).Str("transmissionDelay", time.Since(msg.Timestamp).String()).Int32("RoundID", proofMessage.RoundID).Msg("proof message already processed")
		return nil
	}

	if n.roundProofs.isReplay(proofMessage.RoundID, msg.SentFrom) {
		log.Debug().Str("Player", "Aggregator").Str("Sender", msg.SentFrom).Str("Me", n.Raft.GetHostId()).Int32("RoundID", proofMessage.RoundID).Msg("proof message replayed")
		return nil
	}

	n.storeRoundProofData(proofMessage.RoundID, proofMessage.Proof, msg.SentFrom)

	if len(n.roundProofs.proofs[proofMessage.RoundID]) == n.Raft.SubscribersCount()+1 {
		return n.processCollectedProofs(ctx, proofMessage)
	}

	if len(n.roundProofs.proofs[proofMessage.RoundID]) == 1 {
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

	data, ok := newSubmissionData(n.Name, globalAggregate, proof)
	if !ok {
		return nil
	}
	metrics.SetGlobalAggregate(n.Name, globalAggregate.Value, globalAggregate.Round, globalAggregate.Timestamp)

	// stopped: on shutdown the host closes first, so a round counting only
	// this node's proof (no subscribers left) must not be submitted
	if err := ctx.Err(); err != nil {
		return err
	}
	if n.onSubmission != nil {
		n.onSubmission(data, n.Raft.GetRole() == raft.Leader)
	}
	err := publishSubmissionData(ctx, data)
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

	marshalledTriggerMessage, err := encodeInner(triggerMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal trigger message")
		return err
	}

	message := raft.Message{
		Type:     Trigger,
		SentFrom: n.Raft.GetHostId(),
		Data:     marshalledTriggerMessage,
	}

	return n.Raft.PublishMessage(ctx, message)
}

// PublishPriceDataMessage publishes this node's price for roundId;
// latestRoundId is this node's current RoundID (see PriceDataMessage).
func (n *Aggregator) PublishPriceDataMessage(ctx context.Context, roundId int32, value int64, timestamp time.Time, latestRoundId int32) error {
	priceDataMessage := PriceDataMessage{
		RoundID:       roundId,
		PriceData:     value,
		Timestamp:     timestamp,
		LatestRoundID: latestRoundId,
	}

	marshalledPriceDataMessage, err := encodeInner(priceDataMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal price data message")
		return err
	}

	message := raft.Message{
		Type:     PriceData,
		SentFrom: n.Raft.GetHostId(),
		Data:     marshalledPriceDataMessage,
	}

	return n.Raft.PublishMessage(ctx, message)
}

func (n *Aggregator) PublishPriceFixMessage(ctx context.Context, roundId int32, value int64, timestamp time.Time) error {
	priceFixMessage := PriceFixMessage{
		RoundID:   roundId,
		PriceData: value,
		Timestamp: timestamp,
	}

	marshalledPriceFixMessage, err := encodeInner(priceFixMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal price fix message")
		return err
	}

	message := raft.Message{
		Type:     PriceFix,
		SentFrom: n.Raft.GetHostId(),
		Data:     marshalledPriceFixMessage,
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

	marshalledProofMessage, err := encodeInner(proofMessage)
	if err != nil {
		log.Error().Str("Player", "Aggregator").Err(err).Msg("failed to marshal proof message")
		return err
	}

	message := raft.Message{
		Type:     ProofMsg,
		SentFrom: n.Raft.GetHostId(),
		Data:     marshalledProofMessage,
	}

	return n.Raft.PublishMessage(ctx, message)
}

// pruneRounds keeps the state of rounds from roundHistory-1 before the
// accepted trigger round up to roundLookahead after the current round
// (currentRoundID >= acceptedRoundID) and drops the rest. Anchoring the low
// end on the accepted round keeps a stale trigger's own round locked, as before.
func (n *Aggregator) pruneRounds(acceptedRoundID, currentRoundID int32) {
	// int64: no wrap around near the int32 limits
	low := int64(acceptedRoundID) - (roundHistory - 1)
	high := int64(currentRoundID) + roundLookahead

	n.roundTriggers.pruneOutside(low, high)
	n.roundPrices.pruneOutside(low, high)
	n.roundPriceFixes.pruneOutside(low, high)
	n.roundProofs.pruneOutside(low, high)
}
