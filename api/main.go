package main

import (
	_ "embed"
	"log"

	"bisonai.com/orakl/api/apierr"
	"bisonai.com/orakl/api/blocks"
	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/listener"
	"bisonai.com/orakl/api/proxy"
	"bisonai.com/orakl/api/reporter"
	"bisonai.com/orakl/api/service"
	"bisonai.com/orakl/api/utils"
	"bisonai.com/orakl/api/vrf"

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
	config, err := utils.LoadEnvVars()
	if err != nil {
		panic(err)
	}

	appConfig, err := utils.Setup(version)
	if err != nil {
		panic(err)
	}

	postgres := appConfig.Postgres
	app := appConfig.App

	defer postgres.Close()

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

	apierr.Routes(_router)
	chain.Routes(_router)
	listener.Routes(_router)
	proxy.Routes(_router)
	reporter.Routes(_router)
	service.Routes(_router)
	vrf.Routes(_router)
	blocks.Routes(_router)
}
