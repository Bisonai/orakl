//nolint:all
package reporter

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/reporter"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/libp2p"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const InsertGlobalAggregateQuery = `INSERT INTO global_aggregates (name, value, round) VALUES (@name, @value, @round) RETURNING *`

type TestItems struct {
	app        *App
	reporter   *ReporterNode
	admin      *fiber.App
	messageBus *bus.MessageBus
	tmpData    *TmpData
}

type TmpData struct {
	globalAggregate GlobalAggregate
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)
	tmpGlobalAggregate, err := db.QueryRow[GlobalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"name": "test-aggregate", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate
	return tmpData, nil
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	mb := bus.New(10)
	testItems.messageBus = mb

	admin, err := utils.Setup(utils.SetupInfo{
		Version: "",
		Bus:     mb,
	})
	if err != nil {
		return nil, nil, err
	}

	testItems.admin = admin

	h, ps, err := libp2p.Setup(ctx, "", 10001)
	if err != nil {
		return nil, nil, err
	}

	app := New(mb, *h, ps)
	testItems.app = app

	reporterNode, err := NewNode(ctx, *h, ps)
	if err != nil {
		return nil, nil, err
	}

	testItems.reporter = reporterNode

	tmpData, err := insertSampleData(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.tmpData = tmpData

	v1 := admin.Group("/api/v1")
	reporter.Routes(v1)

	return reporterCleanup(ctx, admin, testItems), testItems, nil

}

func reporterCleanup(ctx context.Context, admin *fiber.App, testItems *TestItems) func() error {
	return func() error {
		err := db.QueryWithoutResult(ctx, "DELETE FROM global_aggregates;", nil)
		if err != nil {
			return err
		}
		err = admin.Shutdown()
		if err != nil {
			return err
		}
		return nil
	}
}

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	code := m.Run()
	db.ClosePool()
	db.CloseRedis()
	os.Exit(code)
}
