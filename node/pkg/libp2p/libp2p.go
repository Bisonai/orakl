package libp2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"crypto/sha256"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/pnet"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/rs/zerolog/log"

	"github.com/multiformats/go-multiaddr"
)

func SetBootNode(ctx context.Context, listenPort int, seed string) (*host.Host, error) {
	var priv crypto.PrivKey
	var err error
	if seed == "" {
		priv, _, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, rand.Reader)
		if err != nil {
			log.Error().Err(err).Msg("Error generating key pair")
			return nil, err
		}
	} else {
		hash := sha256.Sum256([]byte(seed))
		rawKey := ed25519.NewKeyFromSeed(hash[:])
		priv, err = crypto.UnmarshalEd25519PrivateKey(rawKey)
		if err != nil {
			log.Error().Err(err).Msg("Error unmarshalling private key")
			return nil, err
		}
	}

	h, err := makeHost(listenPort, priv)
	if err != nil {
		log.Error().Err(err).Msg("Error creating libp2p host")
		return nil, err
	}

	_ = initDHT(ctx, h, "")

	pi := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}
	fmt.Printf("%s\n", pi.String())
	for _, addr := range pi.Addrs {
		fmt.Println(addr.String() + "/p2p/" + h.ID().String())
	}

	return &h, nil
}

func Setup(ctx context.Context, bootnodeStr string, port int) (*host.Host, *pubsub.PubSub, error) {
	host, err := MakeHost(port)
	if err != nil {
		return nil, nil, err
	}
	ps, err := MakePubsub(ctx, host)
	if err != nil {
		return nil, nil, err
	}

	if bootnodeStr != "" {
		bootnode, bootErr := multiaddr.NewMultiaddr(bootnodeStr)
		if bootErr != nil {
			return nil, nil, bootErr
		}

		peerinfo, bootErr := peer.AddrInfoFromP2pAddr(bootnode)
		if bootErr != nil {
			return nil, nil, bootErr
		}

		bootErr = host.Connect(ctx, *peerinfo)
		if bootErr != nil {
			return nil, nil, bootErr
		}
	}

	discoverString := "orakl-topic-discovery-2024"
	go func() {
		if err = DiscoverPeers(ctx, host, discoverString, bootnodeStr); err != nil {
			log.Error().Err(err).Msg("Error from DiscoverPeers")
		}
	}()

	return &host, ps, nil
}

func GetBootNode(flagNode string) (string, error) {
	var err error
	bootnode := ""

	if flagNode != "" {
		bootnode = flagNode
	}

	if os.Getenv("BOOT_NODE") != "" && bootnode == "" {
		bootnode = os.Getenv("BOOT_NODE")
	}

	if bootnode == "" {
		return "", errors.New("no bootnode specified")
	}

	return bootnode, err
}

func GetListenPort(flagPort int) (int, error) {
	var err error
	listenPort := 0

	if flagPort != 0 {
		listenPort = flagPort
	}

	if os.Getenv("LISTEN_PORT") != "" && listenPort == 0 {
		listenPort, err = strconv.Atoi(os.Getenv("LISTEN_PORT"))
		if err != nil {
			log.Error().Err(err).Msg("Error parsing LISTEN_PORT")
		}
	}

	if os.Getenv("APP_PORT") != "" && listenPort == 0 {
		appPort, err := strconv.Atoi(os.Getenv("APP_PORT"))
		if err != nil {
			log.Error().Err(err).Msg("Error parsing APP_PORT")
		} else {
			listenPort = appPort + 3000
		}
	}

	if listenPort == 0 {
		return 0, errors.New("no libp2p listen port specified")
	}

	return listenPort, nil
}

func MakeHost(listenPort int) (host.Host, error) {
	r := rand.Reader

	log.Debug().Msg("generating private key")
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, r)
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("generating libp2p options")

	return makeHost(listenPort, priv)
}

func makeHost(listenPort int, priv crypto.PrivKey) (host.Host, error) {
	secretString := os.Getenv("PRIVATE_NETWORK_SECRET")
	opts := []libp2p.Option{
		libp2p.EnableNATService(),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
	}

	if secretString != "" {
		hash := sha256.Sum256([]byte(secretString))
		fmt.Println(hash)
		protector := pnet.PSK(hash[:])
		opts = append(opts, libp2p.PrivateNetwork(protector))
	}

	return libp2p.New(opts...)
}

func MakePubsub(ctx context.Context, host host.Host) (*pubsub.PubSub, error) {
	log.Debug().Msg("creating pubsub instance")
	var basePeerFilter pubsub.PeerFilter = func(pid peer.ID, topic string) bool {
		return strings.HasPrefix(pid.String(), "12D")
	}
	var fetcherSubParams = pubsub.DefaultGossipSubParams()

	fetcherSubParams.D = 2
	fetcherSubParams.Dlo = 1
	fetcherSubParams.Dhi = 3

	return pubsub.NewGossipSub(ctx, host, pubsub.WithPeerFilter(basePeerFilter), pubsub.WithGossipSubParams(fetcherSubParams))
}

func GetHostAddress(host host.Host) (string, error) {
	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID()))
	if err != nil {
		log.Error().Err(err).Msg("Error creating multiaddr")
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

	var kademliaDHT *dht.IpfsDHT
	var err error

	if bootstrap == "" {
		log.Info().Msg("No bootstrap provided")
		kademliaDHT, err = dht.New(ctx, h)
		if err != nil {
			panic(err)
		}
	} else {
		log.Info().Msg("Using bootstrap")
		ma, err := multiaddr.NewMultiaddr(bootstrap)
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Got multiaddr")
		info, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			panic(err)
		}

		kademliaDHT, err = dht.New(ctx, h, dht.BootstrapPeers(*info))
		if err != nil {
			panic(err)
		}

		h.Connect(ctx, *info)
		log.Debug().Msg("Connected to bootstrap node")
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting AddrInfo from p2p address")
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Connect(ctx, *peerinfo); err != nil {
				log.Debug().Err(err).Msg("Bootstrap warning")
			}
		}()
	}
	wg.Wait()

	return kademliaDHT
}

func connectToPeers(ctx context.Context, h host.Host, routingDiscovery *drouting.RoutingDiscovery, topicName string) (connected bool, err error) {
	peerChan, err := routingDiscovery.FindPeers(ctx, topicName)
	if err != nil {
		return false, err
	}
	var wg sync.WaitGroup
	for p := range peerChan {
		if p.ID == h.ID() {
			continue // No self connection
		}
		wg.Add(1)
		go func(p peer.AddrInfo) {
			defer wg.Done()
			err := h.Connect(ctx, p)
			if err == nil {
				connected = true
			}
		}(p)
	}
	wg.Wait()
	return connected, nil
}

func DiscoverPeers(ctx context.Context, h host.Host, topicName string, bootstrap string) error {
	kademliaDHT := initDHT(ctx, h, bootstrap)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicName)

	// Look for others who have announced and attempt to connect to them
	var anyConnected bool
	var err error
	for i := 0; i < 10 && !anyConnected; i++ {
		log.Debug().Int("connected peers", len(h.Network().Peers())).Msg("Searching for peers...")
		anyConnected, err = connectToPeers(ctx, h, routingDiscovery, topicName)
		if err != nil {
			return err
		}
		if !anyConnected {
			time.Sleep(time.Second * 3) // wait before retrying
		}
	}
	if !anyConnected {
		return errors.New("no peers connected")
	}
	log.Debug().Msg("Peer discovery complete")
	return nil
}
