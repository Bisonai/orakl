package tests

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

var insertedAdapter adapter.AdapterModel
var insertedFeed feed.FeedModel

func setup() (*fiber.App, error) {
	app, err := utils.Setup("")
	if err != nil {
		return nil, err
	}
	err = insertSampleData(context.Background())
	if err != nil {
		return nil, err
	}
	v1 := app.Group("/api/v1")
	adapter.Routes(v1)
	feed.Routes(v1)
	return app, nil
}

func insertSampleData(ctx context.Context) error {
	tmpAdapter, err := db.QueryRow[adapter.AdapterModel](ctx, adapter.InsertAdapter, map[string]any{"name": "test_adapter"})
	if err != nil {
		return err
	}
	insertedAdapter = tmpAdapter

	tmpFeed, err := db.QueryRow[feed.FeedModel](ctx, adapter.InsertFeed, map[string]any{"name": "test_feed", "adapter_id": insertedAdapter.Id, "definition": `{"test": "test"}`})
	if err != nil {
		return err
	}
	insertedFeed = tmpFeed
	return nil
}

func cleanup() {
	_, err := db.QueryRow[adapter.AdapterModel](context.Background(), adapter.DeleteAdapterById, map[string]any{"id": insertedAdapter.Id})
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	// setup
	code := m.Run()

	db.ClosePool()
	db.CloseRedis()
	// teardown
	os.Exit(code)
}
