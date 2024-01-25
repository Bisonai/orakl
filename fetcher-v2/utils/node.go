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

type Node struct {
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

func NewNode(ctx context.Context, host host.Host, topicString string) (*Node, error) {
	fmt.Println("subscribing to topic:", topicString)

	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &Node{
		Host:  host,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,
		Data:  make(map[string][]int),
	}, nil
}

func (n *Node) Start(ctx context.Context, interval time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	n.Cancel = cancel

	go n.publish(ctx, interval)
	go n.subscribe(ctx)
}

func (n *Node) Stop() {
	if n.Cancel != nil {
		n.Cancel()
	}
}

func (n *Node) publish(ctx context.Context, interval time.Duration) {
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

			n.Data[id] = append(n.Data[id], num)

			fmt.Printf("Publishing %s:%d\n", data.ID, data.Number)

			dataBytes, err := json.Marshal(data)
			if err != nil {
				log.Println(err)
				continue
			}

			err = n.Topic.Publish(ctx, dataBytes)
			if err != nil {

				log.Println(err)
				continue
			}

			executeAtEndOfInterval(start, interval, func() {
				n.calculateAverage(id)
			})
		case <-ctx.Done():
			return
		}
	}
}

func (n *Node) subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:

			m, err := n.Sub.Next(ctx)
			if err != nil {
				log.Fatalln(err)
				continue
			}

			if m.ReceivedFrom == n.Host.ID() {
				//log.Println("Received message from self")
				continue
			}

			var data SampleData
			err = json.Unmarshal(m.Data, &data)
			if err != nil {
				log.Fatalln(err)
				continue
			}
			log.Printf("(%s) Received %s:%d from %s\n", n.Topic.String(), data.ID, data.Number, m.ReceivedFrom)

			n.NodeMutex.Lock()

			n.Data[data.ID] = append(n.Data[data.ID], data.Number)
			n.NodeMutex.Unlock()
		}
	}
}

func (n *Node) calculateAverage(id string) int {
	if len(n.Data) == 0 {
		return 0
	}

	sum := 0
	for _, num := range n.Data[id] {
		sum += num
	}

	fmt.Printf("topic: %s, id: %s, average:%d\n", n.Topic.String(), id, sum/len(n.Data[id]))
	n.Data[id] = nil

	return sum / len(n.Data)
}

func (n *Node) getSubscribersCount() int {
	peers := n.subscribers()
	return len(peers)
}

func (n *Node) subscribers() []peer.ID {
	return n.Ps.ListPeers(n.Topic.String())
}

func (n *Node) PrintInfo() {
	fmt.Println("Node Info:")
	fmt.Println("Host ID:", n.Host.ID())
	fmt.Println("Topic:", n.Topic.String())
	fmt.Println("Data:", n.Data)
}

func (n *Node) String() string {
	return fmt.Sprintf("Host ID: %s, Topic: %s, Data: %v, Subscribers: %d", n.Host.ID(), n.Topic.String(), n.Data, n.getSubscribersCount())
}
