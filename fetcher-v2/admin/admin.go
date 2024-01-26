package admin

import (
	"fmt"
	"log"

	"bisonai.com/fetcher/v2/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type AppContext struct {
	DiscoverString string
	Host           *host.Host
	Nodes          map[string]*utils.FetcherNode
	Peers          map[peer.ID]peer.AddrInfo
}

func Run(port string, appContext AppContext) {
	app := fiber.New()

	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("host", appContext.Host)
		c.Locals("nodes", appContext.Nodes)
		c.Locals("peers", appContext.Peers)
		c.Locals("discoverString", appContext.DiscoverString)
		return c.Next()
	})

	// routes (will be separated later)
	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Orakl fetcherV2 API v1\n")
	})

	v1.Get("/host", func(c *fiber.Ctx) error {
		h := c.Locals("host").(*host.Host)
		hostAddr, err := utils.GetHostAddress(*h)
		if err != nil {
			return err
		}

		return c.SendString(hostAddr)
	})

	v1.Get("/connected-peers", func(c *fiber.Ctx) error {
		h := c.Locals("host").(*host.Host)
		conns := (*h).Network().Conns()
		peers := make([]string, 0, len(conns))
		for _, conn := range conns {
			peers = append(peers, conn.RemotePeer().String())
		}
		return c.JSON(peers)
	})

	v1.Get("/discovered-peers", getAllDiscoveredPeers)

	v1.Get("/node", getAllNodesInfo)
	v1.Get("/node/:topic", getNodeInfo)

	v1.Post("/node/discover", discover)
	v1.Post("/node/dummy", addMultipleDummyNodes)
	v1.Post("/node/start", startAll)
	v1.Post("/node/stop", stopAll)

	v1.Post("/node/:topic", addNode)
	v1.Post("/node/:topic/start", startNode)
	v1.Post("/node/:topic/stop", stopNode)

	err := app.Listen(fmt.Sprintf(":%s", port))
	if err != nil {
		log.Printf("Failed to start server: %v", err)
		return
	}
}
