package tests

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/boot/peer"
	"bisonai.com/orakl/node/pkg/boot/utils"
	"bisonai.com/orakl/node/pkg/db"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/host"
)

type TestItems struct {
	app     *fiber.App
	tmpData *TmpData
	host    host.Host
}

type TmpData struct {
	peer peer.PeerModel
}

func setup(ctx context.Context) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	bootHost, err := libp2pSetup.NewHost(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.host = bootHost

	app, err := utils.Setup(ctx, &bootHost)
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

	tmpPeer, err := db.QueryRow[peer.PeerModel](ctx, peer.InsertPeer, map[string]any{"url": "/ip4/100.78.175.63/tcp/10002/p2p/12D3KooWERrdEepSi8HPRNsfjj3Nd7XcxV9SJcHdovpPLyYUtuch"})
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
		testItems.host.Close()

		return db.QueryWithoutResult(context.Background(), peer.DeletePeerById, map[string]any{"id": testItems.tmpData.peer.ID})
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	db.ClosePool()
	os.Exit(code)
}
