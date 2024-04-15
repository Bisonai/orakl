//nolint:all
package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/admin/aggregator"
	"bisonai.com/orakl/node/pkg/admin/reporter"
	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/db"
	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

const InsertGlobalAggregateQuery = `INSERT INTO global_aggregates (name, value, round) VALUES (@name, @value, @round) RETURNING name, value, round`
const InsertAddressQuery = `INSERT INTO submission_addresses (name, address, interval) VALUES (@name, @address, @interval) RETURNING *`
const TestInterval = 15000

type TestItems struct {
	app        *App
	admin      *fiber.App
	messageBus *bus.MessageBus
	tmpData    *TmpData
}
type TmpData struct {
	globalAggregate   GlobalAggregate
	submissionAddress SubmissionAddress
	proofBytes        []byte
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	signHelper, err := helper.NewSignHelper("")
	if err != nil {
		return nil, err
	}

	key := "globalAggregate:" + "test-aggregate"
	data, err := json.Marshal(map[string]any{"name": "test-aggregate", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	db.Set(ctx, key, string(data), time.Duration(10*time.Second))

	err = db.QueryWithoutResult(ctx, aggregator.InsertAggregator, map[string]any{"name": "test-aggregate"})
	if err != nil {
		return nil, err
	}

	err = db.QueryWithoutResult(ctx, aggregator.InsertAggregator, map[string]any{"name": "test-aggregate2"})
	if err != nil {
		return nil, err
	}

	tmpGlobalAggregate, err := db.QueryRow[GlobalAggregate](ctx, InsertGlobalAggregateQuery, map[string]any{"name": "test-aggregate", "value": int64(15), "round": int64(1)})
	if err != nil {
		return nil, err
	}
	tmpData.globalAggregate = tmpGlobalAggregate

	rawProof, err := signHelper.MakeGlobalAggregateProof(int64(15), time.Now())
	if err != nil {
		return nil, err
	}
	tmpData.proofBytes = bytes.Join([][]byte{rawProof, rawProof}, nil)

	err = db.QueryWithoutResult(ctx, "INSERT INTO proofs (name, round, proof) VALUES (@name, @round, @proof)", map[string]any{"name": "test-aggregate", "round": int64(1), "proof": bytes.Join([][]byte{rawProof, rawProof}, nil)})
	if err != nil {
		return nil, err
	}

	rdbProof := Proof{
		Name:  "test-aggregate",
		Round: int64(1),
		Proof: bytes.Join([][]byte{rawProof, rawProof}, nil),
	}
	rdbProofData, err := json.Marshal(rdbProof)
	if err != nil {
		return nil, err
	}

	err = db.Set(ctx, "proof:test-aggregate|round:1", string(rdbProofData), time.Duration(10*time.Second))
	if err != nil {
		return nil, err
	}

	tmpAddress, err := db.QueryRow[SubmissionAddress](ctx, InsertAddressQuery, map[string]any{"name": "test-aggregate", "address": "0x1234", "interval": TestInterval})
	if err != nil {
		return nil, err
	}
	tmpData.submissionAddress = tmpAddress

	_, err = db.QueryRow[SubmissionAddress](ctx, InsertAddressQuery, map[string]any{"name": "test-aggregate2", "address": "0x1234", "interval": 20000})
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
		err := db.QueryWithoutResult(ctx, "DELETE FROM aggregators;", nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM global_aggregates;", nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM submission_addresses;", nil)
		if err != nil {
			return err
		}

		err = db.QueryWithoutResult(ctx, "DELETE FROM proofs;", nil)
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
