package admin

import (
	"fmt"
	"log"

	"bisonai.com/fetcher/v2/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/host"
)

var nodes []*utils.Node

func Run(port string, h *host.Host) {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("host", h)
		c.Locals("nodes", &nodes)
		return c.Next()
	})

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
		return c.JSON((*h).Network().Peers())
	})

	v1.Post("/node/:topic", addNode)

	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
