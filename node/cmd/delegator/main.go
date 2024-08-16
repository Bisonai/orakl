package main

import (
	"context"
	_ "embed"
	"os"

	"bisonai.com/orakl/node/pkg/logscribeconsumer"

	"bisonai.com/orakl/node/pkg/delegator/contract"
	"bisonai.com/orakl/node/pkg/delegator/function"
	"bisonai.com/orakl/node/pkg/delegator/organization"
	"bisonai.com/orakl/node/pkg/delegator/reporter"
	"bisonai.com/orakl/node/pkg/delegator/sign"
	"bisonai.com/orakl/node/pkg/delegator/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

//go:embed .version
var version string

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("delegator"),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

	err = godotenv.Load()
	if err != nil {
		log.Info().Msg("env file is not found, continuing without .env file")
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
	port = os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	err = app.Listen(":" + port)
	if err != nil {
		panic(err)
	}

}

func SetRouter(router fiber.Router) {
	router.Get("", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Network Delegator")
	})

	contract.Routes(router)
	sign.Routes(router)
	function.Routes(router)
	organization.Routes(router)
	reporter.Routes(router)

}
