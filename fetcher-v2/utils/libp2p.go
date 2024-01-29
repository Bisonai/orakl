package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"

	"github.com/multiformats/go-multiaddr"
)

func MakeHost(listenPort int) (host.Host, error) {
	r := rand.Reader

	log.Println("generating private key")
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, r)
	if err != nil {
		return nil, err
	}
	log.Println("generating libp2p options")

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	log.Println("generating libp2p instance")

	return libp2p.New(opts...)
}

func MakePubsub(ctx context.Context, host host.Host) (*pubsub.PubSub, error) {
	log.Println("generating pubsub instance")
	var basePeerFilter pubsub.PeerFilter = func(pid peer.ID, topic string) bool {
		return strings.HasPrefix(pid.String(), "12D")
	}
	return pubsub.NewGossipSub(ctx, host, pubsub.WithPeerFilter(basePeerFilter))
}

func GetHostAddress(host host.Host) (string, error) {
	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID()))
	if err != nil {
		return "", err
	}

	addr := host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String(), nil
}

func initDHT(ctx context.Context, h host.Host, bootstrap string) *dht.IpfsDHT {
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
	bootstrapPeers := dht.DefaultBootstrapPeers
	if bootstrap != "" {
		// if bootstrap address is specified, add it to first index of the list
		bootstrapPeerAddr, err := multiaddr.NewMultiaddr(bootstrap)
		if err != nil {
			panic(err)
		}
		bootstrapPeers = append([]multiaddr.Multiaddr{bootstrapPeerAddr}, bootstrapPeers...)
	}

	for _, peerAddr := range bootstrapPeers {
		peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			log.Println("Error getting AddrInfo from p2p address:", err)
			continue
		}
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

func DiscoverPeers(ctx context.Context, h host.Host, topicName string, bootstrap string, discoveredPeers map[peer.ID]peer.AddrInfo) {
	kademliaDHT := initDHT(ctx, h, bootstrap)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicName)

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	var wg sync.WaitGroup
	for !anyConnected {
		log.Println("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, topicName)
		if err != nil {
			panic(err)
		}
		for p := range peerChan {
			if p.ID == h.ID() {
				continue // No self connection
			}
			wg.Add(1)
			go func(p peer.AddrInfo) {
				defer wg.Done()
				err := h.Connect(ctx, p)
				if err != nil {
					// log.Printf("Failed connecting to %s, error: %s\n", peer.ID, err)

				} else {
					// log.Println("Connected to:", peer.ID)
					anyConnected = true
					discoveredPeers[p.ID] = p
				}
			}(p)

		}
	}
	wg.Wait()
	log.Println("Peer discovery complete")
}

func ConnectToPeer(ctx context.Context, h host.Host, peerID peer.ID) error {
	// Assume you have a DHT instance in dht
	kademliaDHT := initDHT(ctx, h, "")

	// Find the peer's AddrInfo
	peerInfo, err := kademliaDHT.FindPeer(ctx, peerID)
	if err != nil {
		return fmt.Errorf("failed to find peer: %w", err)
	}

	// Connect to the peer
	if err := h.Connect(ctx, peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer: %w", err)
	}
	log.Println("connected directly to peer:", peerID)

	return nil
}
