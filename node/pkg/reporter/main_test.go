//nolint:all
package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/reporter"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/common/keys"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const InsertGlobalAggregateQuery = `INSERT INTO global_aggregates (config_id, value, round, timestamp) VALUES (@config_id, @value, @round, @timestamp) RETURNING *`
const InsertConfigQuery = `INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval) VALUES (@name, @fetch_interval, @aggregate_interval, @submit_interval) RETURNING name, id, submit_interval, aggregate_interval;`
const TestInterval = 15000

type TestItems struct {
	app        *App
	admin      *fiber.App
	messageBus *bus.MessageBus
	tmpData    *TmpData
}
type TmpData struct {
	globalAggregate GlobalAggregate
	config          Config
	proofBytes      []byte
	proofTime       time.Time
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	signHelper, err := helper.NewSigner(ctx)
	if err != nil {
		return nil, err
	}
	proofTime := time.Now()

	tmpConfig, err := db.QueryRow[Config](ctx, InsertConfigQuery, map[string]any{"name": "test-aggregate", "submit_interval": TestInterval, "fetch_interval": TestInterval, "aggregate_interval": TestInterval})
	if err != nil {
		log.Error().Err(err).Msg("error inserting config 0")
		return nil, err
	}
	tmpData.config = tmpConfig

	err = db.QueryWithoutResult(ctx, InsertConfigQuery, map[string]any{"name": "test-aggregate-2", "submit_interval": TestInterval * 2, "fetch_interval": TestInterval, "aggregate_interval": TestInterval})
	if err != nil {
		log.Error().Err(err).Msg("error inserting config 1")
		return nil, err
	}

	key := keys.GlobalAggregateKey(tmpConfig.ID)
	data, err := json.Marshal(map[string]any{"configId": tmpConfig.ID, "value": int64(15), "round": int64(1), "timestamp": proofTime})
	if err != nil {
		return nil, err
	}
	db.Set(ctx, key, string(data), time.Duration(10*time.Second))

	tmpGlobalAggregate, err := db.QueryRow[GlobalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"config_id": tmpConfig.ID, "value": int64(15), "round": int64(1), "timestamp": proofTime})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate

	rawProof, err := signHelper.MakeGlobalAggregateProof(int64(15), proofTime, "test-aggregate")
	if err != nil {
		return nil, err
	}
	tmpData.proofBytes = bytes.Join([][]byte{rawProof}, nil)
	tmpData.proofTime = proofTime

	err = db.QueryWithoutResult(ctx, "INSERT INTO proofs (config_id, round, proof) VALUES (@config_id, @round, @proof)", map[string]any{"config_id": tmpConfig.ID, "round": int64(1), "proof": bytes.Join([][]byte{rawProof}, nil)})
	if err != nil {
		return nil, err
	}

	rdbProof := Proof{
		ConfigID: tmpConfig.ID,
		Round:    int32(1),
		Proof:    bytes.Join([][]byte{rawProof}, nil),
	}
	rdbProofData, err := json.Marshal(rdbProof)
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, keys.ProofKey(tmpConfig.ID, 1), string(rdbProofData), time.Duration(10*time.Second))
	if err != nil {
		return nil, err
	}

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

	app := New(mb)
	testItems.app = app

	tmpData, err := insertSampleData(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.tmpData = tmpData

	v1 := admin.Group("/api/v1")
	reporter.Routes(v1)

	return reporterCleanup(ctx, admin, app), testItems, nil

}

func reporterCleanup(ctx context.Context, admin *fiber.App, app *App) func() error {
	return func() error {
		err := db.QueryWithoutResult(ctx, "DELETE FROM global_aggregates;", nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM proofs;", nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM configs;", nil)
		if err != nil {
			return err
		}

		err = admin.Shutdown()
		if err != nil {
			return err
		}

		err = app.stopReporters()
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
