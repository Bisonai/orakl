package main

import (
	_ "embed"
	"log"
	"os"

	"bisonai.com/orakl/go-delegator/contract"
	"bisonai.com/orakl/go-delegator/function_"
	"bisonai.com/orakl/go-delegator/organization"
	"bisonai.com/orakl/go-delegator/reporter"
	"bisonai.com/orakl/go-delegator/sign"
	"bisonai.com/orakl/go-delegator/utils"

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
	port = os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	err = app.Listen(":" + port)
	if err != nil {
		panic(err)
	}

}

func SetRouter(_router fiber.Router) {
	_router.Get("", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Network Delegator")
	})

	contract.Routes(_router)
	sign.Routes(_router)
	function_.Routes(_router)
	organization.Routes(_router)
	reporter.Routes(_router)

}
