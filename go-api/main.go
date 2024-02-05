package main

import (
	_ "embed"
	"go-api/adapter"
	"go-api/aggregate"
	"go-api/aggregator"
	"go-api/apierr"
	"go-api/chain"
	"go-api/data"
	"go-api/feed"
	"go-api/l2aggregator"
	"go-api/listener"
	"go-api/proxy"
	"go-api/reporter"
	"go-api/service"
	"go-api/utils"
	"go-api/vrf"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

//go:embed .version
var version string

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("env file is not found, continuing without .env file")
	}
	config := utils.LoadEnvVars()

	appConfig, err := utils.Setup(version)
	if err != nil {
		panic(err)
	}

	postgres := appConfig.Postgres
	redis := appConfig.Redis
	app := appConfig.App

	defer postgres.Close()
	defer redis.Close()

	v1 := app.Group("/api/v1")
	SetRouter(v1)

	var port string
	if val, ok := config["APP_PORT"].(string); ok {
		port = val
	} else {
		port = "3000"
	}

	err = app.Listen(":" + port)
	if err != nil {
		panic(err)
	}
}

func SetRouter(_router fiber.Router) {
	(_router).Get("", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Network API")
	})

	adapter.Routes(_router)
	aggregate.Routes(_router)
	aggregator.Routes(_router)
	apierr.Routes(_router)
	chain.Routes(_router)
	data.Routes(_router)
	feed.Routes(_router)
	l2aggregator.Routes(_router)
	listener.Routes(_router)
	proxy.Routes(_router)
	reporter.Routes(_router)
	service.Routes(_router)
	vrf.Routes(_router)
}
