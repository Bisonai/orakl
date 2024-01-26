package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

/*
healthcheck pubsub node continuously publishes itself and listens for other nodes with same topic string

after receiving message from subscription, following is done:
if message sender is not connected with host, it tries to establish connection with message sender
it removes all other connected peers from host except the peers which sent the message
*/

type HealthCheckerNode struct {
	Host      host.Host
	Ps        *pubsub.PubSub
	Topic     *pubsub.Topic
	Data      map[string]peer.ID
	Sub       *pubsub.Subscription
	NodeMutex sync.Mutex
}

func NewHealthCheckerNode(ctx context.Context, host host.Host, topicString string) (*HealthCheckerNode, error) {
	fmt.Println("subscribing to topic:", topicString)

	var healthCheckerNodeFilter pubsub.PeerFilter = func(pid peer.ID, topic string) bool {
		return topic == topicString && strings.HasPrefix(pid.String(), "12D")
	}

	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithPeerFilter(healthCheckerNodeFilter))
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

	return &HealthCheckerNode{
		Host:  host,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,
		Data:  make(map[string]peer.ID),
	}, nil
}

func (n *HealthCheckerNode) nodeReadiness(rt pubsub.PubSubRouter, topic string) (bool, error) {
	if rt.EnoughPeers(n.Topic.String(), 1) && topic == n.Topic.String() {
		return true, nil
	}
	return false, fmt.Errorf("not enough peers ready")
}

func (n *HealthCheckerNode) Start(ctx context.Context, interval time.Duration) {
	go n.subscribe(ctx)
	go n.publish(ctx, interval)
}

func (n *HealthCheckerNode) publish(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			start := time.Now()
			//log.Println("Publishing ping")
			err := n.Topic.Publish(ctx, []byte("ping"), pubsub.WithReadiness(n.nodeReadiness))
			if err != nil {
				log.Println(err)
			}

			executeAtEndOfInterval(start, interval, func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
				n.refreshConnectedPeers(ctx)
			})
		case <-ctx.Done():
			return
		}
	}
}

func (n *HealthCheckerNode) subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			//log.Println("Stopping health checker node")
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

			//log.Println("Received ping from:", m.ReceivedFrom)

			n.NodeMutex.Lock()
			n.Data[m.ReceivedFrom.String()] = m.ReceivedFrom
			n.NodeMutex.Unlock()
		}
	}
}

func (n *HealthCheckerNode) isConnected(peerID peer.ID) bool {
	for _, conn := range n.Host.Network().Conns() {
		if conn.RemotePeer() == peerID {
			return true
		}
	}
	return false
}

func (n *HealthCheckerNode) refreshConnectedPeers(ctx context.Context) {
	n.NodeMutex.Lock()
	defer n.NodeMutex.Unlock()

	for _, v := range n.Data {
		if !n.isConnected(v) {
			//log.Println("Not connected to peer:", v)
			err := ConnectToPeer(ctx, n.Host, v)
			if err != nil {
				//log.Println("Failed to connect to peer:", err)
			}
		}
	}
	if len(n.Data) > 0 && len(n.Data) < len(n.Host.Network().Conns()) {
		n.removeUndirectNodes()
	}

	n.Data = make(map[string]peer.ID)
}

func (n *HealthCheckerNode) removeUndirectNodes() {
	//log.Println("Removing undirect nodes")
	for _, conn := range n.Host.Network().Conns() {
		peerID := conn.RemotePeer()
		for _, v := range n.Data {
			if peerID != v {
				n.Host.Network().ClosePeer(peerID)
			}
		}
	}
}
