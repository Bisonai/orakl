package tests

import (
	"context"
	"os"
	"testing"

	"bisonai.com/miko/node/pkg/admin/aggregator"
	"bisonai.com/miko/node/pkg/admin/config"
	"bisonai.com/miko/node/pkg/admin/fetcher"
	"bisonai.com/miko/node/pkg/admin/host"
	"bisonai.com/miko/node/pkg/admin/utils"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

type TestItems struct {
	app *fiber.App
	mb  *bus.MessageBus
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.New(10)
	testItems.mb = mb

	app, err := utils.Setup(ctx, utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})

	if err != nil {
		return nil, nil, err
	}

	testItems.app = app

	v1 := app.Group("/api/v1")
	aggregator.Routes(v1)
	fetcher.Routes(v1)
	config.Routes(v1)
	host.Routes(v1)
	return adminCleanup(testItems), testItems, nil
}

func adminCleanup(testItems *TestItems) func() error {
	return func() error {
		return testItems.app.Shutdown()
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	db.CloseRedis()

	os.Exit(code)
}
