package utils

import (
	"context"
	"fmt"
	"time"

	"strings"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/retrier"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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
		return "", errorSentinel.ErrLibP2pEmptyNonLocalAddress
	}

	return addr.Encapsulate(hostAddr).String(), nil
}

func IsHostAlive(ctx context.Context, h host.Host, addr string) (bool, error) {
	info, err := ConnectionUrl2AddrInfo(addr)
	if err != nil {
		return false, err
	}

	lastErr := retrier.Retry(
		func() error {
			log.Info().Str("addr", addr).Str("peer", info.ID.String()).Msg("checking peer alive")
			return h.Connect(ctx, *info)
		},
		3,
		100*time.Millisecond,
		1*time.Second,
	)

	if lastErr != nil {
		return false, errorSentinel.ErrLibP2pFailToConnectPeer
	}

	err = h.Network().ClosePeer(info.ID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func ExtractConnectionUrl(h host.Host) (string, error) {
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
		return "", errorSentinel.ErrLibP2pEmptyNonLocalAddress
	}

	return addr.String() + "/p2p/" + h.ID().String(), nil
}

func ConnectionUrl2AddrInfo(url string) (*peer.AddrInfo, error) {
	peerMultiAddr, err := multiaddr.NewMultiaddr(url)
	if err != nil {
		return nil, err
	}

	info, err := peer.AddrInfoFromP2pAddr(peerMultiAddr)
	if err != nil {
		return nil, err
	}

	return info, nil
}
