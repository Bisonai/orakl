package nodes

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const INTERVAL = time.Second * 10

type ElectorNode struct {
	Host      host.Host
	Ps        *pubsub.PubSub
	Topic     *pubsub.Topic
	Sub       *pubsub.Subscription
	Data      []string
	NodeMutex sync.Mutex
	Cancel    context.CancelFunc
}

func NewElectorNode(host host.Host, ps *pubsub.PubSub, topicString string) (*ElectorNode, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &ElectorNode{
		Host:  host,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,
		Data:  make([]string, 0),
	}, nil
}

func (n *ElectorNode) nodeReadiness(rt pubsub.PubSubRouter, topic string) (bool, error) {
	if topic == n.Topic.String() {
		return true, nil
	}
	return false, fmt.Errorf("not enough peers ready")
}

func (n *ElectorNode) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	n.Cancel = cancel

	go n.subscribe(ctx)
	go n.publish(ctx)
}

func (n *ElectorNode) subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping elector node")
			return
		default:

			m, err := n.Sub.Next(ctx)
			if err != nil {
				log.Println(err)
				continue
			}

			if m.ReceivedFrom == n.Host.ID() {
				continue
			}

			n.NodeMutex.Lock()
			n.Data = append(n.Data, string(m.Message.Data))
			n.NodeMutex.Unlock()
		}
	}
}

func (n *ElectorNode) publish(ctx context.Context) {
	hostId := n.Host.ID().String()
	ticker := time.NewTicker(INTERVAL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			err := n.Topic.Publish(ctx, []byte(hostId), pubsub.WithReadiness(n.nodeReadiness))
			if err != nil {
				log.Println(err)
			}

			executeAtEndOfInterval(start, INTERVAL, func() {
				timestampId := GetIDFromTimestamp(int64(INTERVAL.Seconds()), start)
				isMayer, err := n.amIMayor(timestampId)
				if err != nil {
					log.Println(err)
				}
				if isMayer {

					log.Printf("(%s:%s) I am the mayor", timestampId[len(timestampId)-4:], hostId[len(hostId)-4:])
				}
			})
		case <-ctx.Done():
			return
		}
	}
}

func (n *ElectorNode) amIMayor(id string) (bool, error) {
	n.NodeMutex.Lock()
	defer n.NodeMutex.Unlock()
	defer func() {
		n.Data = nil
	}()

	if len(n.Data) == 0 {
		// elect oneself if no other peers are in the channel
		return true, nil
	}

	n.Data = append(n.Data, n.Host.ID().String())
	sort.Strings(n.Data)
	numberedID, err := strconv.Atoi(id)
	if err != nil {
		return false, err
	}
	mayerIndex := numberedID % len(n.Data)

	if n.Data[mayerIndex] == n.Host.ID().String() {
		return true, nil
	}
	return false, nil
}
