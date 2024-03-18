package utils

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"strings"
	"sync"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/rs/zerolog/log"

	"github.com/multiformats/go-multiaddr"
)

func GetHostAddress(host host.Host) (string, error) {
	hostAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", host.ID()))
	if err != nil {
		log.Error().Err(err).Msg("Error creating multiaddr")
		return "", err
	}

	var addr multiaddr.Multiaddr
	for _, a := range host.Addrs() {
		if strings.Contains(a.String(), "127.0.0.1") {
			continue
		}
		addr = a
		break
	}

	if addr == nil {
		log.Error().Msg("host has no non-local addresses")
		return "", errors.New("host has no non-local addresses")
	}

	return addr.Encapsulate(hostAddr).String(), nil
}

func InitDHT(ctx context.Context, h host.Host, bootstrap string) (*dht.IpfsDHT, error) {
	// Start a DHT, for use in peer discovery. We can't just make a new DHT
	// client because we want each peer to maintain its own local copy of the
	// DHT, so that the bootstrapping node of the DHT can go down without
	// inhibiting future peer discovery.

	var kademliaDHT *dht.IpfsDHT
	var err error

	if bootstrap == "" {
		kademliaDHT, err = dht.New(ctx, h)
		if err != nil {
			log.Error().Err(err).Msg("Error creating DHT without bootstrap")
			return nil, err
		}
	} else {
		var ma multiaddr.Multiaddr
		var info *peer.AddrInfo
		log.Info().Msg("Using bootstrap")
		ma, err = multiaddr.NewMultiaddr(bootstrap)
		if err != nil {
			log.Error().Err(err).Msg("Error creating multiaddr")
			return nil, err
		}
		log.Info().Msg("Got multiaddr")
		info, err = peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			log.Error().Err(err).Msg("Error getting AddrInfo from p2p address")
			return nil, err
		}

		kademliaDHT, err = dht.New(ctx, h, dht.BootstrapPeers(*info))
		if err != nil {
			log.Error().Err(err).Msg("Error creating DHT with bootstrap")
			return nil, err
		}

		err = h.Connect(ctx, *info)
		if err != nil {
			log.Error().Err(err).Msg("Error connecting to bootstrap node")
			return nil, err
		}
		log.Debug().Msg("Connected to bootstrap node")
	}

	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		log.Error().Err(err).Msg("Error bootstrapping DHT")
		return nil, err
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

	return kademliaDHT, nil
}

func connectToPeers(ctx context.Context, h host.Host, routingDiscovery *drouting.RoutingDiscovery, topicName string) (connected bool, err error) {
	peerChan, err := routingDiscovery.FindPeers(ctx, topicName)
	if err != nil {
		return false, err
	}
	successChan := make(chan bool)
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
				successChan <- true
			}
		}(p)
	}
	go func() {
		wg.Wait()
		close(successChan)
	}()

	for success := range successChan {
		if success {
			return true, nil
		}
	}

	return false, nil
}

func DiscoverPeers(ctx context.Context, h host.Host, topicName string, bootstrap string) error {
	kademliaDHT, dhtErr := InitDHT(ctx, h, bootstrap)
	if dhtErr != nil {
		log.Error().Err(dhtErr).Msg("Error initializing DHT")
		return dhtErr
	}
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, topicName)

	// Look for others who have announced and attempt to connect to them
	const maxConnectionTrials = 10
	for i := 0; i < maxConnectionTrials; i++ {
		log.Debug().Int("connected peers", len(h.Network().Peers())).Msg("Searching for peers...")
		connected, err := connectToPeers(ctx, h, routingDiscovery, topicName)
		if err != nil {
			return err
		}
		if !connected {
			time.Sleep(time.Second * 3) // wait before retrying
			continue
		}
		return nil
	}
	return errors.New("no peers connected")
}

func IsHostAlive(ctx context.Context, h host.Host, addr string) (bool, error) {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return false, err
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return false, err
	}

	var lastErr error
	for i := 0; i < 3; i++ { // Retry up to 3 times
		err = h.Connect(ctx, *info)
		if err == nil {
			break
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}

	if lastErr != nil {
		return false, fmt.Errorf("failed to connect to peer")
	}

	err = h.Network().ClosePeer(info.ID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func ExtractPayloadFromHost(h host.Host) (ip string, port int, host_id string, err error) {
	var addr multiaddr.Multiaddr
	for _, a := range h.Addrs() {
		if strings.Contains(a.String(), "127.0.0.1") {
			continue
		}
		addr = a
		break
	}

	if addr == nil {
		log.Error().Msg("host has no non-local addresses")
		return "", 0, "", errors.New("host has no non-local addresses")
	}

	splitted := strings.Split(addr.String(), "/")
	if len(splitted) < 5 {
		log.Error().Msg("error splitting address")
		return "", 0, "", errors.New("error splitting address")
	}
	ip = splitted[2]
	rawPort := splitted[4]
	port, err = strconv.Atoi(rawPort)
	if err != nil {
		log.Error().Err(err).Msg("error converting port to int")
		return "", 0, "", err
	}

	return ip, port, h.ID().String(), nil
}
