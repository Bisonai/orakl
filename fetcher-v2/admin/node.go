package admin

import (
	"strconv"
	"time"

	"bisonai.com/fetcher/v2/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

func discover(c *fiber.Ctx) error {
	h, err := utils.GetHost(c)
	if err != nil {
		log.Panicf("failed to load host: %s", err)
	}

	s, err := utils.GetDiscoverString(c)
	if err != nil {
		log.Panicf("failed to load discover string: %s", err)
	}

	go utils.DiscoverPeers(c.Context(), *h, s)

	return c.SendString("triggered peer discovery")
}

func getAllNodesInfo(c *fiber.Ctx) error {
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Panicf("failed to load nodes: %s", err)
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
		log.Panicf("failed to load node: %s", err)
	}

	return c.JSON(node.String())
}

func addNode(c *fiber.Ctx) error {
	h, err := utils.GetHost(c)
	if err != nil {
		log.Panicf("failed to load host: %s", err)
	}

	topicString := c.Params("topic")
	node, err := utils.NewNode(c.Context(), *h, topicString)
	if err != nil {
		log.Panicf("failed to create node: %s", err)
	}

	err = utils.SetNode(c, topicString, node)
	if err != nil {
		log.Panicf("failed to set node: %s", err)
	}

	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Panicf("failed to load nodes: %s", err)
	}

	info := make([]string, 0, len(nodes))
	// Iterate over the nodes and call PrintInfo on each one
	for _, node := range nodes {
		info = append(info, node.String())
	}

	// Return the info slice as the JSON response
	return c.JSON(info)
}

func startNode(c *fiber.Ctx) error {
	topicString := c.Params("topic")
	node, err := utils.GetNode(c, topicString)
	if err != nil {
		log.Panicf("failed to load node: %s", err)
	}
	node.Start(c.Context(), time.Second*2)
	return c.SendString("node(" + topicString + ") started")
}

func stopNode(c *fiber.Ctx) error {
	topicString := c.Params("topic")
	node, err := utils.GetNode(c, topicString)
	if err != nil {
		log.Panicf("failed to load node: %s", err)
	}
	node.Stop()
	return c.SendString("node(" + topicString + ") stopped")
}

func addMultipleDummyNodes(c *fiber.Ctx) error {
	for i := 0; i < 10; i++ {
		topicString := "orakl-test-topic-" + strconv.Itoa(i)
		h, err := utils.GetHost(c)
		if err != nil {
			log.Panicf("failed to load host: %s", err)
		}

		node, err := utils.NewNode(c.Context(), *h, topicString)
		if err != nil {
			log.Panicf("failed to create node: %s", err)
		}

		err = utils.SetNode(c, topicString, node)
		if err != nil {
			log.Panicf("failed to set node: %s", err)
		}
	}
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Panicf("failed to load nodes: %s", err)
	}

	info := make([]string, 0, len(nodes))
	// Iterate over the nodes and call PrintInfo on each one
	for _, node := range nodes {
		info = append(info, node.String())
	}

	// Return the info slice as the JSON response
	return c.JSON(info)
}

func startAll(c *fiber.Ctx) error {
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Panicf("failed to load nodes: %s", err)
	}

	for _, node := range nodes {
		if node.Cancel != nil {
			continue
		}
		node.Start(c.Context(), time.Second*5)
	}

	return c.SendString("all nodes started")
}

func stopAll(c *fiber.Ctx) error {
	nodes, err := utils.GetNodes(c)
	if err != nil {
		log.Panicf("failed to load nodes: %s", err)
	}

	for _, node := range nodes {
		node.Stop()
	}

	return c.SendString("all nodes stopped")
}
