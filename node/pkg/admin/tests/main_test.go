package tests

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

type TestItems struct {
	app      *fiber.App
	mb       *bus.MessageBus
	tempData *TempData
}

type TempData struct {
	adapter adapter.AdapterModel
	feed    feed.FeedModel
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.NewMessageBus()
	testItems.mb = mb

	app, err := utils.Setup(utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})

	if err != nil {
		return nil, nil, err
	}

	testItems.app = app

	tempData, err := insertSampleData(ctx)
	if err != nil {
		return nil, nil, err
	}

	testItems.tempData = tempData

	v1 := app.Group("/api/v1")
	adapter.Routes(v1)
	feed.Routes(v1)
	fetcher.Routes(v1)
	return cleanup(testItems), testItems, nil
}

func insertSampleData(ctx context.Context) (*TempData, error) {
	var tempData = new(TempData)

	tmpAdapter, err := db.QueryRow[adapter.AdapterModel](ctx, adapter.InsertAdapter, map[string]any{"name": "test_adapter"})
	if err != nil {
		return nil, err
	}
	tempData.adapter = tmpAdapter

	tmpFeed, err := db.QueryRow[feed.FeedModel](ctx, adapter.InsertFeed, map[string]any{"name": "test_feed", "adapter_id": tmpAdapter.Id, "definition": `{"test": "test"}`})
	if err != nil {
		return nil, err
	}
	tempData.feed = tmpFeed

	return tempData, nil
}

func cleanup(testItems *TestItems) func() error {
	return func() error {
		err := testItems.app.Shutdown()
		if err != nil {
			return err
		}
		_, err = db.QueryRow[adapter.AdapterModel](context.Background(), adapter.DeleteAdapterById, map[string]any{"id": testItems.tempData.adapter.Id})
		return err
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()

	os.Exit(code)
}
