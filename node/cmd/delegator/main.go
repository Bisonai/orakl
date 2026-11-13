package main

import (
	"context"
	_ "embed"
	"fmt"
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

	if len(os.Args) > 1 && os.Args[1] == "token" {
		if err := runToken(context.Background(), os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "allow" {
		if err := runAllow(context.Background(), os.Args[2:], os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	loginit.InitZeroLog()

	appConfig, err := utils.Setup("0.0.1")
	if err != nil {
		panic(err)
	}

	app := appConfig.App
	v1 := app.Group("/api/v1")
	if appConfig.Store != nil {
		defer appConfig.Store.Close()
		SetSigningRouter(v1)
	} else {
		defer appConfig.Postgres.Close()
		SetRouter(v1)
	}

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

// SetSigningRouter registers the SQLite-mode routes: no allowlist or
// transaction management API (use 'allow' on the CLI).
func SetSigningRouter(router fiber.Router) {
	router.Get("", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Network Delegator")
	})
	sign.SigningRoutes(router)
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
