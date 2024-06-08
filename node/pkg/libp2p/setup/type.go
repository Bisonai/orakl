package setup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"

	"bisonai.com/orakl/node/pkg/secrets"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/rs/zerolog/log"
)

type HostConfig struct {
	Port         int
	PrivateKey   crypto.PrivKey
	SecretString string
	HolePunch    bool
	Quic         bool
}

type HostOption func(*HostConfig)

func WithPort(port int) HostOption {
	return func(hc *HostConfig) {
		hc.Port = port
	}
}

func WithPrivateKey(priv crypto.PrivKey) HostOption {
	return func(hc *HostConfig) {
		hc.PrivateKey = priv
	}
}

func WithSecretString(secretString string) HostOption {
	return func(hc *HostConfig) {
		hc.SecretString = secretString
	}
}

func WithHolePunch() HostOption {
	return func(hc *HostConfig) {
		hc.HolePunch = true
	}
}

func WithQuic() HostOption {
	return func(hc *HostConfig) {
		hc.Quic = true
	}
}

func NewHost(ctx context.Context, opts ...HostOption) (host.Host, error) {
	defaultPort := 0
	defaultPortStr := os.Getenv("LISTEN_PORT")
	if defaultPortStr != "" {
		tmp, err := strconv.Atoi(defaultPortStr)
		if err == nil {
			defaultPort = tmp
		}
	}

	config := &HostConfig{
		Port:         defaultPort,
		PrivateKey:   nil,
		SecretString: secrets.GetSecret("PRIVATE_NETWORK_SECRET"),
		HolePunch:    false,
		Quic:         false,
	}
	for _, opt := range opts {
		opt(config)
	}

	if config.PrivateKey == nil {
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, err
		}
		config.PrivateKey = priv
	}

	listenStr := ""
	if config.Quic {
		listenStr = fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", config.Port)
	} else {
		listenStr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.Port)
	}

	libp2pOpts := []libp2p.Option{
		libp2p.Identity(config.PrivateKey),
		libp2p.ListenAddrStrings(listenStr),
	}

	if config.SecretString != "" {
		hash := sha256.Sum256([]byte(config.SecretString))
		protector := pnet.PSK(hash[:])
		libp2pOpts = append(libp2pOpts, libp2p.PrivateNetwork(protector))
	}

	if config.HolePunch {
		libp2pOpts = append(libp2pOpts, libp2p.EnableHolePunching())
	}

	h, err := libp2p.New(libp2pOpts...)
	return h, err
}

func MakePubsub(ctx context.Context, host host.Host) (*pubsub.PubSub, error) {
	log.Debug().Msg("creating pubsub instance")
	return pubsub.NewGossipSub(ctx, host)
}
