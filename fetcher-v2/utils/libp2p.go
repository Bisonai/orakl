package utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"

	ma "github.com/multiformats/go-multiaddr"
)

type Node struct {
	Host     host.Host
	Ps       *pubsub.PubSub
	Topic    *pubsub.Topic
	Sub      *pubsub.Subscription
	Numbers  map[string][]int
	NumMutex sync.Mutex
	Cancel   context.CancelFunc
}

type SampleData struct {
	Number int    `json:"number"`
	ID     string `json:"id"`
}

func MakeHost() (host.Host, error) {
	var r io.Reader
	r = rand.Reader

	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	return libp2p.New(opts...)
}

func GetHostAddress(host host.Host) (string, error) {
	hostAddr, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID()))
	if err != nil {
		return "", err
	}

	addr := host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String(), nil
}

func initDHT(ctx context.Context, h host.Host) *dht.IpfsDHT {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.
	kademliaDHT, err := dht.New(ctx, h)
	if err != nil {
		panic(err)
	}
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				log.Println("Bootstrap warning:", err)
			}
		}()
	}
	wg.Wait()

	return kademliaDHT
}

func DiscoverPeers(ctx context.Context, h host.Host, topicName string) {
	kademliaDHT := initDHT(ctx, h)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicName)

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	for !anyConnected {
		// log.Println("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, topicName)
		if err != nil {
			panic(err)
		}
		for peer := range peerChan {
			if peer.ID == h.ID() {
				continue // No self connection
			}
			err := h.Connect(ctx, peer)
			if err != nil {
				// log.Printf("Failed connecting to %s, error: %s\n", peer.ID, err)

			} else {
				// log.Println("Connected to:", peer.ID)
				anyConnected = true
			}
		}
	}
	log.Println("Peer discovery complete")
}

// node is a struct that holds the host, pubsub, topic, and subscription
// each node for each datafeed

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
		Host:    host,
		Ps:      ps,
		Topic:   topic,
		Sub:     sub,
		Numbers: make(map[string][]int),
	}, nil
}

func (n *Node) Start(ctx context.Context, interval time.Duration) {
	ctx, cancel := context.WithCancel(ctx)
	n.Cancel = cancel

	go n.publishRandomNumbers(ctx, interval)
	go n.collectNumbers(ctx)
}

func (n *Node) Stop() {
	if n.Cancel != nil {
		n.Cancel()
	}
}

func (n *Node) publishRandomNumbers(ctx context.Context, interval time.Duration) {
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

			n.Numbers[id] = append(n.Numbers[id], num)

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

func (n *Node) collectNumbers(ctx context.Context) {
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
				continue
			}

			var data SampleData
			err = json.Unmarshal(m.Data, &data)
			if err != nil {
				log.Fatalln(err)
				continue
			}
			fmt.Printf("Received %s:%d from %s\n", data.ID, data.Number, m.ReceivedFrom)

			n.NumMutex.Lock()

			n.Numbers[data.ID] = append(n.Numbers[data.ID], data.Number)
			n.NumMutex.Unlock()
		}
	}
}

func (n *Node) calculateAverage(id string) int {
	if len(n.Numbers) == 0 {
		return 0
	}

	sum := 0
	for _, num := range n.Numbers[id] {
		sum += num
	}

	fmt.Printf("id: %s, average:%d\n", id, sum/len(n.Numbers[id]))

	return sum / len(n.Numbers)
}

func (n *Node) calculateMajority(id string) int {
	if len(n.Numbers[id]) == 0 {
		return 0
	}
	counts := make(map[int]int)
	for _, num := range n.Numbers[id] {
		counts[num]++
	}
	majorityNum := n.Numbers[id][0]
	maxCount := 0
	for num, count := range counts {
		if count > maxCount {
			maxCount = count
			majorityNum = num
		}
	}
	return majorityNum
}

func (n *Node) clearID(id string) {
	n.Numbers[id] = nil
}

func (n *Node) getSubscribersCount() int {
	peers := n.Ps.ListPeers(n.Topic.String())
	return len(peers)
}

func (n *Node) getSubscribers() []peer.ID {
	return n.Ps.ListPeers(n.Topic.String())
}

func executeAtEndOfInterval(start time.Time, interval time.Duration, function func()) {
	elapsed := time.Since(start)
	remaining := interval - elapsed

	if remaining > 0 {
		<-time.After(remaining)
		function()
	}
}
