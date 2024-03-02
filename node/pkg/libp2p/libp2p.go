package libp2p

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

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
	"github.com/rs/zerolog/log"

	"github.com/multiformats/go-multiaddr"
)

func SetBootNode(listenPort int) (*host.Host, error) {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Error().Err(err).Msg("Error generating key pair")
		return nil, err
	}

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/"+strconv.Itoa(listenPort)), libp2p.Identity(priv))
	if err != nil {
		log.Error().Err(err).Msg("Error creating libp2p host")
		return nil, err
	}

	_, err = dht.New(context.Background(), h)
	if err != nil {
		log.Error().Err(err).Msg("Error creating DHT")
		return nil, err
	}

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

func Setup(ctx context.Context) (*host.Host, *pubsub.PubSub, error) {
	flagBootnode := flag.String("b", "", "bootnode address")
	flagPort := flag.Int("p", 0, "libp2p port")
	flag.Parse()

	port, err := getListenPort(*flagPort)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get listen port")
		return nil, nil, err
	}
	host, err := MakeHost(port)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create libp2p host")
		return nil, nil, err
	}
	ps, err := MakePubsub(ctx, host)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create pubsub")
		return nil, nil, err
	}

	bootnodeStr, _ := getBootNode(*flagBootnode)

	if bootnodeStr != "" {
		log.Debug().Str("bootnode", bootnodeStr).Msg("connecting to bootnode")
		bootnode, bootErr := multiaddr.NewMultiaddr(bootnodeStr)
		if bootErr != nil {
			log.Fatal().Err(bootErr).Msg("Failed to create multiaddr")
			return nil, nil, bootErr
		}

		peerinfo, bootErr := peer.AddrInfoFromP2pAddr(bootnode)
		if bootErr != nil {
			log.Fatal().Err(bootErr).Msg("Failed to create peerinfo")
			return nil, nil, bootErr
		}

		bootErr = host.Connect(ctx, *peerinfo)
		if bootErr != nil {
			log.Fatal().Err(bootErr).Msg("Failed to connect to bootnode")
			return nil, nil, bootErr
		}
		log.Debug().Str("bootnode", bootnodeStr).Msg("connected to bootnode")
	}

	go func() {
		if err = DiscoverPeers(ctx, host, "orakl-test-discover-2024", bootnodeStr); err != nil {
			log.Error().Err(err).Msg("Error from DiscoverPeers")
		}
	}()

	return &host, ps, nil
}

func getBootNode(flagNode string) (string, error) {
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

func getListenPort(flagPort int) (int, error) {
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

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}
	log.Debug().Msg("generating libp2p instance")

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

func DiscoverPeers(ctx context.Context, h host.Host, topicName string, bootstrap string) error {
	kademliaDHT := initDHT(ctx, h, bootstrap)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicName)

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	var wg sync.WaitGroup
	for !anyConnected {
		log.Debug().Msg("Searching for peers...")
		peerChan, err := routingDiscovery.FindPeers(ctx, topicName)
		if err != nil {
			return err
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
					// log.Trace().Msg("Failed connecting to " + p.ID.String())
				} else {
					// log.Trace().Str("connectedTo", p.ID.String()).Msg("Connected to peer")
					anyConnected = true
				}
			}(p)
		}
	}
	wg.Wait()
	if !anyConnected {
		return errors.New("no peers connected")
	}
	log.Debug().Msg("Peer discovery complete")
	return nil
}
