package aggregator

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/raft"
	"bisonai.com/orakl/node/pkg/utils"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const RoundSync raft.MessageType = "roundSync"
const PriceData raft.MessageType = "priceData"

type Aggregator struct {
	Raft  *raft.Raft
	Host  host.Host
	Ps    *pubsub.PubSub
	Topic *pubsub.Topic
	Sub   *pubsub.Subscription

	LeaderJobTicker *time.Ticker
	JobTicker       *time.Ticker

	LeaderJobTimeout *time.Duration
	JobTimeout       *time.Duration

	CollectedPrices map[int][]int
	AggregatorMutex sync.Mutex

	RoundID int
}

type RoundSyncMessage struct {
	LeaderID string `json:"leaderID"`
	RoundID  int    `json:"roundID"`
}

type PriceDataMessage struct {
	RoundID   int `json:"roundID"`
	PriceData int `json:"priceData"`
}

func NewAggregator(h host.Host, ps *pubsub.PubSub, topicString string) (*Aggregator, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	leaderTimeout := 5 * time.Second

	aggregator := Aggregator{
		Raft:  raft.NewRaftNode(100), // consider updating after testing
		Host:  h,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,

		LeaderJobTimeout: &leaderTimeout,
		JobTimeout:       nil,

		CollectedPrices: map[int][]int{},
		AggregatorMutex: sync.Mutex{},
	}

	return &aggregator, nil
}

func (a *Aggregator) Run() {
	a.Raft.Run(a)
}

func (a *Aggregator) GetSub() *pubsub.Subscription {
	return a.Sub
}

func (a *Aggregator) GetPubSub() *pubsub.PubSub {
	return a.Ps
}

func (a *Aggregator) GetHost() host.Host {
	return a.Host
}

func (a *Aggregator) GetTopic() *pubsub.Topic {
	return a.Topic
}

func (a *Aggregator) GetLeaderJobTimeout() *time.Duration {
	return a.LeaderJobTimeout
}

func (a *Aggregator) GetLeaderJobTicker() *time.Ticker {
	return a.LeaderJobTicker
}

func (a *Aggregator) SetLeaderJobTicker(d *time.Duration) error {
	if d == nil {
		a.LeaderJobTicker = nil
		return nil
	}
	a.LeaderJobTicker = time.NewTicker(*d)
	return nil
}

func (a *Aggregator) GetJobTimeout() *time.Duration {
	return a.JobTimeout
}

func (a *Aggregator) GetJobTicker() *time.Ticker {
	return a.JobTicker
}

func (a *Aggregator) SetJobTicker(d *time.Duration) error {
	if d == nil {
		a.JobTicker = nil
		return nil
	}
	a.JobTicker = time.NewTicker(*d)
	return nil
}

func (a *Aggregator) Job() error {
	return nil
}

func (a *Aggregator) LeaderJob() error {
	// leader continously sends roundId in regular basis and triggers all other nodes to run its job
	a.RoundID++
	roundMessage := RoundSyncMessage{
		LeaderID: a.Host.ID().String(),
		RoundID:  a.RoundID,
	}

	marshalledRoundMessage, err := json.Marshal(roundMessage)
	if err != nil {
		return err
	}

	message := raft.Message{
		Type:     RoundSync,
		SentFrom: a.Host.ID().String(),
		Data:     json.RawMessage(marshalledRoundMessage),
	}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return a.Topic.Publish(context.Background(), data)
}

func (a *Aggregator) HandleCustomMessage(message raft.Message) error {
	switch message.Type {
	case RoundSync:
		a.HandleRoundSyncMessage(message)
		// every node runs its job when leader sends roundSync message
	case PriceData:
		a.HandlePriceDataMessage(message)
	}
	return nil
}

func (a *Aggregator) HandleRoundSyncMessage(msg raft.Message) error {
	var roundSyncMessage RoundSyncMessage
	err := json.Unmarshal(msg.Data, &roundSyncMessage)
	if err != nil {
		return err
	}
	a.RoundID = roundSyncMessage.RoundID

	// pull latest local aggregate and send to peers
	latestAggregate := utils.RandomNumberGenerator()
	priceDataMessage := PriceDataMessage{
		RoundID:   a.RoundID,
		PriceData: latestAggregate,
	}
	marshalledPriceDataMessage, err := json.Marshal(priceDataMessage)
	if err != nil {
		return err
	}
	message := raft.Message{
		Type:     PriceData,
		SentFrom: a.Host.ID().String(),
		Data:     json.RawMessage(marshalledPriceDataMessage),
	}

	log.Println("publishing price data message: ", latestAggregate)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return a.Topic.Publish(context.Background(), data)
}

func (a *Aggregator) HandlePriceDataMessage(msg raft.Message) error {
	var priceDataMessage PriceDataMessage
	err := json.Unmarshal(msg.Data, &priceDataMessage)
	if err != nil {
		return err
	}
	a.AggregatorMutex.Lock()
	defer a.AggregatorMutex.Unlock()
	if _, ok := a.CollectedPrices[priceDataMessage.RoundID]; !ok {
		a.CollectedPrices[priceDataMessage.RoundID] = []int{}
	}
	a.CollectedPrices[priceDataMessage.RoundID] = append(a.CollectedPrices[priceDataMessage.RoundID], priceDataMessage.PriceData)
	if len(a.CollectedPrices[priceDataMessage.RoundID]) >= len(a.Ps.ListPeers(a.Topic.String()))+1 {
		// handle aggregation here once all the data have been collected
		median := utils.FindMedian(a.CollectedPrices[priceDataMessage.RoundID])
		roundID := strconv.Itoa(priceDataMessage.RoundID)
		aggregate := strconv.Itoa(median)
		log.Println("Aggregate for round:", roundID, "is:", aggregate)
		delete(a.CollectedPrices, priceDataMessage.RoundID)
	}
	return nil
}
