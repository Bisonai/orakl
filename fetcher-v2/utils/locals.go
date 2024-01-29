package utils

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

func GetDiscoverString(c *fiber.Ctx) (string, error) {
	s, ok := c.Locals("discoverString").(string)
	if !ok {
		return s, errors.New("failed to load discover string")
	}
	return s, nil
}

func GetHost(c *fiber.Ctx) (*host.Host, error) {
	h, ok := c.Locals("host").(*host.Host)
	if !ok {
		return h, errors.New("failed to load host")
	}
	return h, nil
}

func GetPubsub(c *fiber.Ctx) (*pubsub.PubSub, error) {
	ps, ok := c.Locals("pubsub").(*pubsub.PubSub)
	if !ok {
		return ps, errors.New("failed to load pubsub")
	}
	return ps, nil
}

func GetNodes(c *fiber.Ctx) (map[string]*FetcherNode, error) {
	nodes, ok := c.Locals("nodes").(map[string]*FetcherNode)
	if !ok {
		return nodes, errors.New("failed to load nodes")
	}
	return nodes, nil
}

func GetNode(c *fiber.Ctx, topic string) (*FetcherNode, error) {
	nodes, err := GetNodes(c)
	if err != nil {
		return nil, err
	}
	node, ok := nodes[topic]
	if !ok {
		return node, errors.New("failed to load node")
	}
	return node, nil
}

func SetNode(c *fiber.Ctx, topic string, node *FetcherNode) error {
	nodes, err := GetNodes(c)
	if err != nil {
		return err
	}
	nodes[topic] = node
	return nil
}

func GetPeers(c *fiber.Ctx) (map[peer.ID]peer.AddrInfo, error) {
	peers, ok := c.Locals("peers").(map[peer.ID]peer.AddrInfo)
	if !ok {
		return peers, errors.New("failed to load peers")
	}
	return peers, nil
}

func GetPeer(c *fiber.Ctx, peerID peer.ID) (peer.AddrInfo, error) {
	peers, err := GetPeers(c)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	peer, ok := peers[peerID]
	if !ok {
		return peer, errors.New("failed to load peer")
	}
	return peer, nil
}

func SetPeer(c *fiber.Ctx, peerID peer.ID, peer peer.AddrInfo) error {
	peers, err := GetPeers(c)
	if err != nil {
		return err
	}
	peers[peerID] = peer
	return nil
}
