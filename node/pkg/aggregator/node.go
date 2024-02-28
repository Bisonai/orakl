package aggregator

import (
	"context"
	"encoding/json"
	"sync"

	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

func NewNode(h host.Host, ps *pubsub.PubSub, topicString string) (*AggregatorNode, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	leaderTimeout := 5 * time.Second

	aggregator := AggregatorNode{
		Raft: raft.NewRaftNode(h, ps, topic, sub, 100), // consider updating after testing

		LeaderJobTimeout: &leaderTimeout,
		JobTimeout:       nil,

		CollectedPrices: map[int][]int{},
		AggregatorMutex: sync.Mutex{},
	}

	return &aggregator, nil
}

func (n *AggregatorNode) Run(ctx context.Context) {
	n.Raft.Run(ctx, n)
}

func (n *AggregatorNode) GetLeaderJobTimeout() *time.Duration {
	return n.LeaderJobTimeout
}

func (n *AggregatorNode) GetLeaderJobTicker() *time.Ticker {
	return n.LeaderJobTicker
}

func (n *AggregatorNode) SetLeaderJobTicker(d *time.Duration) error {
	if d == nil {
		n.LeaderJobTicker = nil
		return nil
	}
	n.LeaderJobTicker = time.NewTicker(*d)
	return nil
}

func (n *AggregatorNode) LeaderJob() error {
	// leader continously sends roundId in regular basis and triggers all other nodes to run its job
	n.RoundID++
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

func (n *AggregatorNode) HandleCustomMessage(message raft.Message) error {
	switch message.Type {
	case RoundSync:
		return n.HandleRoundSyncMessage(message)
		// every node runs its job when leader sends roundSync message
	case PriceData:
		return n.HandlePriceDataMessage(message)
	}
	return nil
}

/*
should be updated further later to handle various edge cases
1. leader's roundSync message could be lower than follower's roundId
-> might need to add another phase where all the peers agree on the roundId to use

2. roundId should be stored and loaded from db on node restarts
-> should carefully handle when it should be stored and loaded
*/
func (n *AggregatorNode) HandleRoundSyncMessage(msg raft.Message) error {
	var roundSyncMessage RoundSyncMessage
	err := json.Unmarshal(msg.Data, &roundSyncMessage)
	if err != nil {
		return err
	}
	n.RoundID = roundSyncMessage.RoundID

	// pull latest local aggregate and send to peers
	latestAggregate := utils.RandomNumberGenerator()
	priceDataMessage := PriceDataMessage{
		RoundID:   n.RoundID,
		PriceData: latestAggregate,
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

func (n *AggregatorNode) HandlePriceDataMessage(msg raft.Message) error {
	var priceDataMessage PriceDataMessage
	err := json.Unmarshal(msg.Data, &priceDataMessage)
	if err != nil {
		return err
	}
	n.AggregatorMutex.Lock()
	defer n.AggregatorMutex.Unlock()
	if _, ok := n.CollectedPrices[priceDataMessage.RoundID]; !ok {
		n.CollectedPrices[priceDataMessage.RoundID] = []int{}
	}
	n.CollectedPrices[priceDataMessage.RoundID] = append(n.CollectedPrices[priceDataMessage.RoundID], priceDataMessage.PriceData)
	if len(n.CollectedPrices[priceDataMessage.RoundID]) >= len(n.Raft.Ps.ListPeers(n.Raft.Topic.String()))+1 {
		// handle aggregation here once all the data have been collected
		median := utils.FindMedian(n.CollectedPrices[priceDataMessage.RoundID])
		roundID := strconv.Itoa(priceDataMessage.RoundID)
		aggregate := strconv.Itoa(median)
		log.Debug().Msg("Aggregate for round: " + roundID + " is: " + aggregate)
		delete(n.CollectedPrices, priceDataMessage.RoundID)
	}
	return nil
}
