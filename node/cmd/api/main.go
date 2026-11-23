package main

import (
	_ "embed"

	"bisonai.com/miko/node/pkg/api/apierr"
	"bisonai.com/miko/node/pkg/api/blocks"
	"bisonai.com/miko/node/pkg/api/chain"
	"bisonai.com/miko/node/pkg/api/listener"
	"bisonai.com/miko/node/pkg/api/proxy"
	"bisonai.com/miko/node/pkg/api/reporter"
	"bisonai.com/miko/node/pkg/api/service"
	"bisonai.com/miko/node/pkg/api/utils"
	"bisonai.com/miko/node/pkg/api/vrf"
	"bisonai.com/miko/node/pkg/utils/buildinfo"
	"bisonai.com/miko/node/pkg/utils/loginit"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Info().Msg("env file is not found, continuing without .env file")
	}

	loginit.InitZeroLog()
	buildinfo.Register("api")

	config, err := utils.LoadEnvVars()
	if err != nil {
		panic(err)
	}

	appConfig, err := utils.Setup("0.0.1")
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
		return c.SendString(buildinfo.Root("Orakl Network API"))
	})
	(_router).Get("/version", buildinfo.FiberHandler)

	apierr.Routes(_router)
	chain.Routes(_router)
	listener.Routes(_router)
	proxy.Routes(_router)
	reporter.Routes(_router)
	service.Routes(_router)
	vrf.Routes(_router)
	blocks.Routes(_router)
}
