package admin

import (
	"time"

	"bisonai.com/orakl/node/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func discover(c *fiber.Ctx) error {
	h, err := utils.GetHost(c)
	if err != nil {
		log.Errorf("failed to load host: %s", err)
	}

	s, err := utils.GetDiscoverString(c)
	if err != nil {
		log.Errorf("failed to load discover string: %s", err)
	}

	discoveredPeers, err := utils.GetPeers(c)
	if err != nil {
		log.Errorf("failed to load discovered peers: %s", err)
	}

	go utils.DiscoverPeers(c.Context(), *h, s, "", discoveredPeers)

	return c.SendString("triggered peer discovery")
}

func getAllDiscoveredPeers(c *fiber.Ctx) error {
	peers, err := utils.GetPeers(c)
	if err != nil {
		log.Errorf("failed to load peers: %s", err)
	}

	peerList := make([]string, 0, len(peers))
	for _, peer := range peers {
		peerList = append(peerList, peer.String())
	}

	return c.JSON(peerList)
}

func getAllNodesInfo(c *fiber.Ctx) error {
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Errorf("failed to load nodes: %s", err)
	}

	info := make([]string, 0, len(nodes))
	// Iterate over the nodes and call PrintInfo on each one
	for _, node := range nodes {
		info = append(info, node.String())
	}

	// Return the info slice as the JSON response
	return c.JSON(info)
}

func getNodeInfo(c *fiber.Ctx) error {
	topicString := c.Params("topic")
	node, err := utils.GetNode(c, topicString)
	if err != nil {
		log.Errorf("failed to load node: %s", err)
	}

	return c.JSON(node.String())
}

func addNode(c *fiber.Ctx) error {

	h, err := utils.GetHost(c)
	if err != nil {
		log.Errorf("failed to load host: %s", err)
	}

	ps, err := utils.GetPubsub(c)
	if err != nil {
		log.Errorf("failed to load pubsub: %s", err)
	}

	topicString := c.Params("topic")

	node, err := utils.NewNode(*h, ps, topicString)
	if err != nil {
		log.Errorf("failed to create node: %s", err)
	}

	err = utils.SetNode(c, topicString, node)
	if err != nil {
		log.Errorf("failed to set node: %s", err)
	}

	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Errorf("failed to load nodes: %s", err)
	}

	info := make([]string, 0, len(nodes))

	for _, node := range nodes {
		info = append(info, node.String())
	}

	return c.JSON(info)
}

func startNode(c *fiber.Ctx) error {
	topicString := c.Params("topic")
	if topicString == "" {
		log.Errorf("topic string is empty")
	}
	node, err := utils.GetNode(c, topicString)
	if err != nil {
		log.Errorf("failed to load node: %s", err)
	}

	node.Start(time.Second * 2)
	return c.SendString("node(" + topicString + ") started")
}

func stopNode(c *fiber.Ctx) error {
	topicString := c.Params("topic")
	node, err := utils.GetNode(c, topicString)
	if err != nil {
		log.Errorf("failed to load node: %s", err)
	}
	node.Stop()
	return c.SendString("node(" + topicString + ") stopped")
}

func startAll(c *fiber.Ctx) error {
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Errorf("failed to load nodes: %s", err)
	}

	for _, node := range nodes {
		if node.Cancel != nil {
			continue
		}
		node.Start(time.Second * 5)
	}

	return c.SendString("all nodes started")
}

func stopAll(c *fiber.Ctx) error {
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Errorf("failed to load nodes: %s", err)
	}

	for _, node := range nodes {
		node.Stop()
	}

	return c.SendString("all nodes stopped")
}
