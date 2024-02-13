package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/beevik/ntp"
	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type PubSubComponents struct {
	Ps    *pubsub.PubSub
	Topic *pubsub.Topic
	Sub   *pubsub.Subscription
}

type NodeData struct {
	NextRound       string
	SuggestedRounds []string
	Prices          map[string][]int
	Mutex           sync.Mutex
}

type FetcherNode struct {
	Host           host.Host
	PubSub         PubSubComponents
	Data           NodeData
	NextRoundReady chan bool
	Cancel         context.CancelFunc
}

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type RoundData struct {
	Suggestion string `json:"suggestion"`
}

type PriceData struct {
	Number int    `json:"number"`
	ID     string `json:"id"`
}

func NewNode(host host.Host, ps *pubsub.PubSub, topicString string) (*FetcherNode, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &FetcherNode{
		Host:           host,
		PubSub:         PubSubComponents{Ps: ps, Topic: topic, Sub: sub},
		Data:           NodeData{NextRound: "", SuggestedRounds: []string{}, Prices: make(map[string][]int), Mutex: sync.Mutex{}},
		NextRoundReady: make(chan bool, 1),
		Cancel:         nil,
	}, nil
}

func (n *FetcherNode) nodeReadiness(rt pubsub.PubSubRouter, topic string) (bool, error) {
	if rt.EnoughPeers(n.PubSub.Topic.String(), 1) && topic == n.PubSub.Topic.String() {
		return true, nil
	}
	return false, fmt.Errorf("not enough peers ready")
}

func (n *FetcherNode) Start(interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	n.Cancel = cancel

	go n.subscribe(ctx)

	var baseTime time.Time
	ntpTime, err := ntp.Time("pool.ntp.org")
	if err != nil {
		log.Println("ntp time failed:" + err.Error())
		baseTime = time.Now()
	} else {
		baseTime = ntpTime
	}
	startTime := baseTime.Truncate(interval).Add(interval)
	durationUntilStart := time.Until(startTime)
	// synchronized start for message publish
	time.AfterFunc(durationUntilStart, func() {
		go n.publish(ctx, interval)
	})
}

func (n *FetcherNode) Stop() {
	if n.Cancel != nil {
		log.Println("stopping node")
		n.Cancel()
	}
}

func (n *FetcherNode) suggestRound(ctx context.Context, t time.Time, interval time.Duration) (string, error) {
	hostId := n.Host.ID().String()
	suggestingId := GetIDFromTimestamp(int64(interval.Seconds()), t)

	roundData := RoundData{
		Suggestion: suggestingId,
	}

	marshalledRoundData, err := json.Marshal(roundData)
	if err != nil {
		return "", err
	}
	sendRoundMessage := Message{
		Type: "round",
		Data: json.RawMessage(marshalledRoundData),
	}

	roundDataBytes, err := json.Marshal(sendRoundMessage)
	if err != nil {
		return "", err
	}
	err = n.PubSub.Topic.Publish(ctx, roundDataBytes, pubsub.WithReadiness(n.nodeReadiness))
	if err != nil {
		return "", err
	}
	log.Printf("(%s) suggested round: %s\n", hostId[len(hostId)-4:], suggestingId)

	return suggestingId, nil
}

func (n *FetcherNode) publishPrice(ctx context.Context, nextRound string) error {
	hostId := n.Host.ID().String()
	num := RandomNumberGenerator()
	priceData := PriceData{
		Number: num,
		ID:     nextRound,
	}

	marshalledPriceData, err := json.Marshal(priceData)
	if err != nil {
		return err
	}

	sendMessage := Message{
		Type: "price",
		Data: json.RawMessage(marshalledPriceData),
	}

	dataBytes, err := json.Marshal(sendMessage)
	if err != nil {
		return err
	}

	err = n.PubSub.Topic.Publish(ctx, dataBytes, pubsub.WithReadiness(n.nodeReadiness))

	if err != nil {
		return err
	}

	log.Printf("(%s) published %s:%d\n", hostId[len(hostId)-4:], nextRound, num)

	return nil
}

func (n *FetcherNode) publish(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			_, err := n.suggestRound(ctx, t, interval)
			if err != nil {
				log.Println("suggest round failed:" + err.Error())
				continue
			}

			// wait until next round id is determined
			n.determineNextRoundID()
			n.Data.Mutex.Lock()
			nextRound := n.Data.NextRound
			n.Data.Mutex.Unlock()

			err = n.publishPrice(ctx, nextRound)
			if err != nil {
				log.Println("publish price failed:" + err.Error())
				continue
			}
		case <-ctx.Done():
			log.Println("stopping publish")
			return
		}
	}
}

func (n *FetcherNode) handlePriceMessage(ctx context.Context, message Message) error {
	priceData, err := n.unmarshalPrice(message.Data)
	if err != nil {
		return err
	}
	log.Printf("(%s) Received %s:%d\n", n.PubSub.Topic.String(), priceData.ID, priceData.Number)
	n.Data.Mutex.Lock()
	defer n.Data.Mutex.Unlock()
	n.Data.Prices[priceData.ID] = append(n.Data.Prices[priceData.ID], priceData.Number)

	if len(n.Data.Prices[priceData.ID]) > n.getSubscribersCount() {
		log.Println("calculating average for:" + priceData.ID)
		err := n.calculateAverage(priceData.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *FetcherNode) handleRoundMessage(ctx context.Context, message Message) error {
	roundData, err := n.unmarshalRound(message.Data)
	if err != nil {
		return err
	}
	n.Data.Mutex.Lock()
	defer n.Data.Mutex.Unlock()
	n.Data.SuggestedRounds = append(n.Data.SuggestedRounds, roundData.Suggestion)
	if len(n.Data.SuggestedRounds) > n.getSubscribersCount() {
		n.Data.NextRound, err = getMaxFromStringSlice(n.Data.SuggestedRounds)
		if err != nil {
			return err
		}
		log.Println("Next round:", n.Data.NextRound)
		n.Data.SuggestedRounds = []string{}
		n.NextRoundReady <- true
	}
	return nil
}

func (n *FetcherNode) subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("stopping subscribe")
			return
		default:
			rawMessage, err := n.PubSub.Sub.Next(ctx)
			if err != nil {
				log.Println("message receive failed:" + err.Error())
				continue
			}

			message, err := n.unmarshalMessage(rawMessage.Data)
			if err != nil {
				log.Println("unexpected message:" + err.Error())
				continue
			}

			switch message.Type {
			case "price":
				err := n.handlePriceMessage(ctx, message)
				if err != nil {
					log.Println("handle price message failed:" + err.Error())
				}
			case "round":
				err := n.handleRoundMessage(ctx, message)
				if err != nil {
					log.Println("handle round message failed:" + err.Error())
				}
			default:
				log.Println("unexpected message type")
			}
		}
	}
}

func (n *FetcherNode) unmarshalMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}
	return m, nil
}

func (n *FetcherNode) unmarshalPrice(data json.RawMessage) (PriceData, error) {
	var p PriceData
	err := json.Unmarshal(data, &p)
	if err != nil {
		return PriceData{}, err
	}
	return p, nil
}

func (n *FetcherNode) unmarshalRound(data json.RawMessage) (RoundData, error) {
	var r RoundData
	err := json.Unmarshal(data, &r)
	if err != nil {
		return RoundData{}, err
	}
	return r, nil
}

func (n *FetcherNode) calculateAverage(id string) error {
	if len(n.Data.Prices) == 0 {
		return fmt.Errorf("no data to calculate average")
	}

	sum := 0
	for _, num := range n.Data.Prices[id] {
		sum += num
	}

	result := sum / len(n.Data.Prices[id])
	if len(n.Data.Prices[id]) > 1 {
		fmt.Printf("topic: %s, id: %s, average:%d\n", n.PubSub.Topic.String(), id, result)
	}

	delete(n.Data.Prices, id)
	return nil
}

func (n *FetcherNode) getSubscribersCount() int {
	peers := n.subscribers()
	return len(peers)
}

func (n *FetcherNode) subscribers() []peer.ID {
	return n.PubSub.Ps.ListPeers(n.PubSub.Topic.String())
}

func (n *FetcherNode) PrintInfo() {
	n.Data.Mutex.Lock()
	defer n.Data.Mutex.Unlock()
	fmt.Println("Node Info:")
	fmt.Println("Host ID:", n.Host.ID())
	fmt.Println("Topic:", n.PubSub.Topic.String())
	fmt.Println("Prices:", n.Data.Prices)
}

func (n *FetcherNode) String() string {
	return fmt.Sprintf("Host ID: %s, Topic: %s, Prices: %v, Subscribers: %d", n.Host.ID(), n.PubSub.Topic.String(), n.Data.Prices, n.getSubscribersCount())
}

func (n *FetcherNode) determineNextRoundID() {
	for len(n.NextRoundReady) > 0 { // Drain the channel
		<-n.NextRoundReady
	}
	<-n.NextRoundReady
}
