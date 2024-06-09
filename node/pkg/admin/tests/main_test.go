package tests

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/config"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/providerUrl"
	"bisonai.com/orakl/node/pkg/admin/proxy"
	"bisonai.com/orakl/node/pkg/admin/reporter"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/admin/wallet"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/encryptor"
	"github.com/gofiber/fiber/v2"
)

type TestItems struct {
	app     *fiber.App
	mb      *bus.MessageBus
	tmpData *TmpData
}

type TmpData struct {
	config      config.ConfigModel
	feed        feed.FeedModel
	proxy       proxy.ProxyModel
	wallet      wallet.WalletModel
	providerUrl providerUrl.ProviderUrlModel
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.New(10)
	testItems.mb = mb

	app, err := utils.Setup(utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})

	if err != nil {
		return nil, nil, err
	}

	testItems.app = app

	tmpData, err := insertSampleData(ctx)
	if err != nil {
		return nil, nil, err
	}

	testItems.tmpData = tmpData

	v1 := app.Group("/api/v1")
	aggregator.Routes(v1)
	feed.Routes(v1)
	fetcher.Routes(v1)
	proxy.Routes(v1)
	wallet.Routes(v1)
	reporter.Routes(v1)
	providerUrl.Routes(v1)
	config.Routes(v1)
	return adminCleanup(testItems), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpConfig, err := db.QueryRow[config.ConfigModel](ctx, "INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval) VALUES (@name,  @fetch_interval, @aggregate_interval, @submit_interval) RETURNING *;", map[string]any{"name": "test_config", "fetch_interval": 1, "aggregate_interval": 1, "submit_interval": 1})
	if err != nil {
		return nil, err
	}
	tmpData.config = tmpConfig

	tmpFeed, err := db.QueryRow[feed.FeedModel](ctx, "INSERT INTO feeds (name, config_id, definition) VALUES (@name, @config_id, @definition) RETURNING *;", map[string]any{"name": "test_feed", "config_id": tmpConfig.ID, "definition": `{"test": "test"}`})
	if err != nil {
		return nil, err
	}
	tmpData.feed = tmpFeed

	tmpProxy, err := db.QueryRow[proxy.ProxyModel](ctx, proxy.InsertProxy, map[string]any{"protocol": "http", "host": "localhost", "port": 80, "location": "test"})
	if err != nil {
		return nil, err
	}
	tmpData.proxy = tmpProxy

	encryptedTestPk, err := encryptor.EncryptText("0xec5e03e43eb58d0ea03c639e9bf1894793091928f1e222ce18df961ad4efb04e")
	if err != nil {
		return nil, err
	}
	tmpWallet, err := db.QueryRow[wallet.WalletModel](ctx, wallet.InsertWallet, map[string]any{"pk": encryptedTestPk})
	if err != nil {
		return nil, err
	}
	tmpWallet.Pk = "0xec5e03e43eb58d0ea03c639e9bf1894793091928f1e222ce18df961ad4efb04e"
	tmpData.wallet = tmpWallet

	tmpProviderUrl, err := db.QueryRow[providerUrl.ProviderUrlModel](ctx, providerUrl.InsertProviderUrl, map[string]any{"chain_id": 1, "url": "test_url", "priority": 1})
	if err != nil {
		return nil, err
	}
	tmpData.providerUrl = tmpProviderUrl

	return tmpData, nil
}

func adminCleanup(testItems *TestItems) func() error {
	return func() error {
		err := testItems.app.Shutdown()
		if err != nil {
			return err
		}

		_, err = db.QueryRow[proxy.ProxyModel](context.Background(), proxy.DeleteProxyById, map[string]any{"id": testItems.tmpData.proxy.ID})
		if err != nil {
			return err
		}

		_, err = db.QueryRow[providerUrl.ProviderUrlModel](context.Background(), providerUrl.DeleteProviderUrlById, map[string]any{"id": testItems.tmpData.providerUrl.ID})
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(context.Background(), "DELETE FROM configs", nil)
		if err != nil {
			return err
		}

		return db.QueryWithoutResult(context.Background(), wallet.DeleteWalletById, map[string]any{"id": testItems.tmpData.wallet.ID})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()

	os.Exit(code)
}
