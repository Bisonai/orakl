package setup

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"

	"crypto/sha256"
	"strings"

	"bisonai.com/orakl/node/pkg/libp2p/utils"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/rs/zerolog/log"

	"github.com/multiformats/go-multiaddr"
)

type BootPeerModel struct {
	Id     int64  `db:"id" json:"id"`
	Ip     string `db:"ip" json:"ip"`
	Port   int    `db:"port" json:"port"`
	HostId string `db:"host_id" json:"host_id"`
}

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

	_, err = utils.InitDHT(ctx, h, "")
	if err != nil {
		log.Error().Err(err).Msg("Error initializing DHT")
		return nil, err
	}

	pi := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

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
		if err = utils.DiscoverPeers(ctx, host, discoverString, bootnodeStr); err != nil {
			log.Error().Err(err).Msg("Error from DiscoverPeers")
		}
	}()

	return &host, ps, nil
}

func SetupFromBootApi(ctx context.Context, port int) (host.Host, *pubsub.PubSub, error) {
	host, err := MakeHost(port)
	if err != nil {
		log.Error().Err(err).Msg("Error making host")
		return nil, nil, err
	}

	ps, err := MakePubsub(ctx, host)
	if err != nil {
		log.Error().Err(err).Msg("Error making pubsub")
		return nil, nil, err
	}

	ip, port, hostId, err := utils.ExtractPayloadFromHost(host)
	if err != nil {
		log.Error().Err(err).Msg("Error extracting payload from host")
		return nil, nil, err
	}

	apiEndpoint := os.Getenv("BOOT_API_URL")
	if apiEndpoint == "" {
		log.Info().Msg("boot api endpoint not set, using default url: http://localhost:8089")
		apiEndpoint = "http://localhost:8089"
	}

	dbPeers, err := request.UrlRequest[[]BootPeerModel](apiEndpoint+"/api/v1/peer/sync", "POST", map[string]any{
		"ip":      ip,
		"port":    port,
		"host_id": hostId,
	}, nil, "")
	if err != nil {
		log.Error().Err(err).Msg("Error getting peers from boot API")
		return nil, nil, err
	}

	for _, dbPeer := range dbPeers {
		peerAddr := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", dbPeer.Ip, dbPeer.Port, dbPeer.HostId)
		peerMultiAddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Error().Err(err).Msg("Error creating multiaddr")
			return nil, nil, err
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(peerMultiAddr)
		if err != nil {
			log.Error().Err(err).Msg("Error getting AddrInfo from p2p address")
			return nil, nil, err
		}

		err = host.Connect(ctx, *peerInfo)
		if err != nil {
			log.Error().Err(err).Msg("error connecting to peer")
			return nil, nil, err
		}
	}

	return host, ps, nil
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
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
	}

	secretString := os.Getenv("PRIVATE_NETWORK_SECRET")
	if secretString != "" {
		hash := sha256.Sum256([]byte(secretString))
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

	return pubsub.NewGossipSub(ctx, host, pubsub.WithPeerFilter(basePeerFilter))
}
