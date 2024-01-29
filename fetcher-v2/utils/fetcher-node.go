package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type FetcherNode struct {
	Host      host.Host
	Ps        *pubsub.PubSub
	Topic     *pubsub.Topic
	Sub       *pubsub.Subscription
	Data      map[string][]int
	NodeMutex sync.Mutex
	Cancel    context.CancelFunc
}

type SampleData struct {
	Number int    `json:"number"`
	ID     string `json:"id"`
}

func NewNode(host host.Host, ps *pubsub.PubSub, topicString string) (*FetcherNode, error) {
	var fetcherSubParams = pubsub.DefaultGossipSubParams()

	fetcherSubParams.D = 2
	fetcherSubParams.Dlo = 1
	fetcherSubParams.Dhi = 3

	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &FetcherNode{
		Host:  host,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,
		Data:  make(map[string][]int),
	}, nil
}

func (n *FetcherNode) nodeReadiness(rt pubsub.PubSubRouter, topic string) (bool, error) {
	if rt.EnoughPeers(n.Topic.String(), 1) && topic == n.Topic.String() {
		return true, nil
	}
	return false, fmt.Errorf("not enough peers ready")
}

func (n *FetcherNode) Start(interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	n.Cancel = cancel

	go n.subscribe(ctx)
	go n.publish(ctx, interval)
}

func (n *FetcherNode) Stop() {
	if n.Cancel != nil {
		log.Println("stopping node")
		n.Cancel()
	}
}

func (n *FetcherNode) publish(ctx context.Context, interval time.Duration) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:

			start := time.Now()
			num := RandomNumberGenerator()
			id := GetIDFromTimestamp(int64(interval.Seconds()), t)

			data := SampleData{
				Number: num,
				ID:     id,
			}

			n.NodeMutex.Lock()
			n.Data[id] = append(n.Data[id], num)
			n.NodeMutex.Unlock()

			dataBytes, err := json.Marshal(data)
			if err != nil {
				log.Println("json marshal failed:" + err.Error())
				continue
			}

			err = n.Topic.Publish(ctx, dataBytes, pubsub.WithReadiness(n.nodeReadiness))
			// err = n.Topic.Publish(ctx, dataBytes)
			if err != nil {
				log.Println("publish failed:" + err.Error())
				continue
			}
			hostId := n.Host.ID().String()
			log.Printf("(%s) published %s:%d\n", hostId[len(hostId)-4:], data.ID, data.Number)

			executeAtEndOfInterval(start, interval, func() {
				err := n.calculateAverage(id)
				if err != nil {
					log.Println(err.Error())
				}
			})
		case <-ctx.Done():
			log.Println("stoppping publish")
			return
		}
	}
}

func (n *FetcherNode) subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("stopping subscribe")
			return
		default:
			m, err := n.Sub.Next(ctx)
			if err != nil {
				log.Println("message recieve failed:" + err.Error())
				continue
			}

			if m.ReceivedFrom == n.Host.ID() {
				// log.Println("Received message from self")
				continue
			}

			recievedFrom := m.ReceivedFrom

			var data SampleData
			err = json.Unmarshal(m.Data, &data)
			if err != nil {
				log.Println("json unmarshal failed" + err.Error())
				continue
			}
			log.Printf("(%s) Received %s:%d from %s\n", n.Topic.String(), data.ID, data.Number, recievedFrom[len(recievedFrom)-4:])

			n.NodeMutex.Lock()
			n.Data[data.ID] = append(n.Data[data.ID], data.Number)
			n.NodeMutex.Unlock()
		}
	}
}

func (n *FetcherNode) calculateAverage(id string) error {
	n.NodeMutex.Lock()
	defer n.NodeMutex.Unlock()
	if len(n.Data) == 0 {
		return fmt.Errorf("no data to calculate average")
	}

	sum := 0
	for _, num := range n.Data[id] {
		sum += num
	}

	result := sum / len(n.Data[id])
	if len(n.Data[id]) > 1 {
		fmt.Printf("topic: %s, id: %s, average:%d\n", n.Topic.String(), id, result)
	}

	delete(n.Data, id)
	return nil
}

func (n *FetcherNode) getSubscribersCount() int {
	peers := n.subscribers()
	return len(peers)
}

func (n *FetcherNode) subscribers() []peer.ID {
	return n.Ps.ListPeers(n.Topic.String())
}

func (n *FetcherNode) PrintInfo() {
	n.NodeMutex.Lock()
	defer n.NodeMutex.Unlock()
	fmt.Println("Node Info:")
	fmt.Println("Host ID:", n.Host.ID())
	fmt.Println("Topic:", n.Topic.String())
	fmt.Println("Data:", n.Data)
}

func (n *FetcherNode) String() string {
	return fmt.Sprintf("Host ID: %s, Topic: %s, Data: %v, Subscribers: %d", n.Host.ID(), n.Topic.String(), n.Data, n.getSubscribersCount())
}
