package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/adapter"
	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/admin/fetcher"
	"bisonai.com/orakl/node/pkg/admin/proxy"
	"bisonai.com/orakl/node/pkg/admin/reporter"
	"bisonai.com/orakl/node/pkg/admin/submissionAddress"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/admin/wallet"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

type TestItems struct {
	app     *fiber.App
	mb      *bus.MessageBus
	tmpData *TmpData
}

type TmpData struct {
	aggregator        aggregator.AggregatorModel
	adapter           adapter.AdapterModel
	submissionAddress submissionAddress.SubmissionAddressModel
	feed              feed.FeedModel
	proxy             proxy.ProxyModel
	wallet            wallet.WalletModel
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
	adapter.Routes(v1)
	feed.Routes(v1)
	fetcher.Routes(v1)
	proxy.Routes(v1)
	wallet.Routes(v1)
	reporter.Routes(v1)
	submissionAddress.Routes(v1)

	return adminCleanup(testItems), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpAdapter, err := db.QueryRow[adapter.AdapterModel](ctx, adapter.InsertAdapter, map[string]any{"name": "test_adapter"})
	if err != nil {
		return nil, err
	}
	tmpData.adapter = tmpAdapter

	tmpAggregator, err := db.QueryRow[aggregator.AggregatorModel](ctx, aggregator.InsertAggregator, map[string]any{"name": "test_aggregator"})
	if err != nil {
		return nil, err
	}
	tmpData.aggregator = tmpAggregator

	tmpFeed, err := db.QueryRow[feed.FeedModel](ctx, adapter.InsertFeed, map[string]any{"name": "test_feed", "adapter_id": tmpAdapter.Id, "definition": `{"test": "test"}`})
	if err != nil {
		return nil, err
	}
	tmpData.feed = tmpFeed

	tmpProxy, err := db.QueryRow[proxy.ProxyModel](ctx, proxy.InsertProxy, map[string]any{"protocol": "http", "host": "localhost", "port": 80, "location": "test"})
	if err != nil {
		return nil, err
	}
	tmpData.proxy = tmpProxy

	tmpWallet, err := db.QueryRow[wallet.WalletModel](ctx, wallet.InsertWallet, map[string]any{"pk": "test_pk"})
	if err != nil {
		return nil, err
	}
	tmpData.wallet = tmpWallet

	tmpSubmissionAddress, err := db.QueryRow[submissionAddress.SubmissionAddressModel](ctx, submissionAddress.InsertSubmissionAddress, map[string]any{"name": "test_submission_address", "address": "test_address", "interval": sql.NullInt32{Valid: false}})
	if err != nil {
		return nil, err
	}
	tmpData.submissionAddress = tmpSubmissionAddress

	return tmpData, nil
}

func adminCleanup(testItems *TestItems) func() error {
	return func() error {
		err := testItems.app.Shutdown()
		if err != nil {
			return err
		}
		_, err = db.QueryRow[adapter.AdapterModel](context.Background(), adapter.DeleteAdapterById, map[string]any{"id": testItems.tmpData.adapter.Id})
		if err != nil {
			return err
		}

		_, err = db.QueryRow[aggregator.AggregatorModel](context.Background(), aggregator.DeleteAggregatorById, map[string]any{"id": testItems.tmpData.aggregator.Id})
		if err != nil {
			return err
		}

		_, err = db.QueryRow[proxy.ProxyModel](context.Background(), proxy.DeleteProxyById, map[string]any{"id": testItems.tmpData.proxy.Id})
		if err != nil {
			return err
		}

		_, err = db.QueryRow[submissionAddress.SubmissionAddressModel](context.Background(), submissionAddress.DeleteSubmissionAddressById, map[string]any{"id": testItems.tmpData.submissionAddress.Id})
		if err != nil {
			return err
		}

		return db.QueryWithoutResult(context.Background(), wallet.DeleteWalletById, map[string]any{"id": testItems.tmpData.wallet.Id})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()

	os.Exit(code)
}
