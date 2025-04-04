package main

import (
	_ "embed"
	"os"

	"bisonai.com/miko/node/pkg/delegator/contract"
	"bisonai.com/miko/node/pkg/delegator/function"
	"bisonai.com/miko/node/pkg/delegator/organization"
	"bisonai.com/miko/node/pkg/delegator/reporter"
	"bisonai.com/miko/node/pkg/delegator/sign"
	"bisonai.com/miko/node/pkg/delegator/utils"
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
