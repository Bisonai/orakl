package admin

import (
	"bisonai.com/fetcher/v2/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/libp2p/go-libp2p/core/host"
)

func addNode(c *fiber.Ctx) error {
	h := c.Locals("host").(*host.Host)
	topicString := c.Params("topic")
	node, err := utils.NewNode(c.Context(), *h, topicString)
	if err != nil {
		log.Panicf("failed to create node: %s", err)
	}
	nodes := c.Locals("nodes").(*[]*utils.Node)
	*nodes = append(*nodes, node)

	// Create a slice to store the PrintInfo results
	info := make([]string, len(*nodes))

	// Iterate over the nodes and call PrintInfo on each one
	for i, node := range *nodes {
		info[i] = node.String()
	}

	// Return the info slice as the JSON response
	return c.JSON(info)
}
