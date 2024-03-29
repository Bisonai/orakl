//nolint:all
package reporter

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/reporter"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const InsertGlobalAggregateQuery = `INSERT INTO global_aggregates (name, value, round) VALUES (@name, @value, @round) RETURNING name, value, round`
const InsertAddressQuery = `INSERT INTO submission_addresses (name, address) VALUES (@name, @address) RETURNING *`

type TestItems struct {
	app        *App
	admin      *fiber.App
	messageBus *bus.MessageBus
	tmpData    *TmpData
}
type TmpData struct {
	globalAggregate   GlobalAggregate
	submissionAddress SubmissionAddress
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	key := "globalAggregate:" + "test-aggregate"
	data, err := json.Marshal(map[string]any{"value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	db.Set(ctx, key, string(data), time.Duration(10*time.Second))

	tmpGlobalAggregate, err := db.QueryRow[GlobalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"name": "test-aggregate", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate

	tmpAddress, err := db.QueryRow[SubmissionAddress](ctx, InsertAddressQuery, map[string]any{"name": "test-aggregate", "address": "0x1234"})
	if err != nil {
		return nil, err
	}
	tmpData.submissionAddress = tmpAddress

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

	h, err := libp2p_setup.MakeHost(10001)
	if err != nil {
		return nil, nil, err
	}

	ps, err := libp2p_setup.MakePubsub(ctx, h)
	if err != nil {
		return nil, nil, err
	}

	app := New(mb, h, ps)
	testItems.app = app

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

		err = db.QueryWithoutResult(ctx, "DELETE FROM submission_addresses;", nil)
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
