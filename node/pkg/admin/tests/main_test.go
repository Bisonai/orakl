package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/host"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/testutil/testenv"
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
	stopRedis, err := testenv.StartRedis()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	db.CloseRedis()
	stopRedis()

	os.Exit(code)
}
