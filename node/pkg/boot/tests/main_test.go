package tests

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/boot/peer"
	"bisonai.com/orakl/node/pkg/boot/utils"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/gofiber/fiber/v2"
)

type TestItems struct {
	app     *fiber.App
	tmpData *TmpData
}

type TmpData struct {
	peer peer.PeerModel
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	app, err := utils.Setup(ctx)
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
	peer.Routes(v1)

	return bootCleanup(testItems), testItems, nil
}

func insertSampleData(ctx context.Context) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpPeer, err := db.QueryRow[peer.PeerModel](ctx, peer.InsertPeer, map[string]any{"ip": "127.0.0.1", "port": 10000, "host_id": "12DGKooWM8vWWqGPWWNCVPqb4tfqGrzx45W257GDBSeYbDSSLabc"})
	if err != nil {
		return nil, err
	}
	tmpData.peer = tmpPeer
	return tmpData, nil
}

func bootCleanup(testItems *TestItems) func() error {

	return func() error {
		err := testItems.app.Shutdown()
		if err != nil {
			return err
		}

		return db.QueryWithoutResult(context.Background(), peer.DeletePeerById, map[string]any{"id": testItems.tmpData.peer.ID})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	db.ClosePool()
	os.Exit(code)
}
