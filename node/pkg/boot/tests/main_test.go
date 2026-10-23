package tests

import (
	"context"
	"path/filepath"
	"testing"

	"bisonai.com/miko/node/pkg/boot/peer"
	"bisonai.com/miko/node/pkg/boot/utils"
	"bisonai.com/miko/node/pkg/boot/utils/peerstore"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/host"
)

type TestItems struct {
	app     *fiber.App
	tmpData *TmpData
	host    host.Host
	store   *peerstore.Store
}

type TmpData struct {
	peer peer.PeerModel
}

func setup(ctx context.Context, t *testing.T) (func() error, *TestItems, error) {
	var testItems = new(TestItems)

	store, err := peerstore.Open(ctx, filepath.Join(t.TempDir(), "boot.db"))
	if err != nil {
		return nil, nil, err
	}
	testItems.store = store

	bootHost, err := libp2pSetup.NewHost(ctx)
	if err != nil {
		return nil, nil, err
	}
	testItems.host = bootHost

	app, err := utils.Setup(ctx, &bootHost, store)
	if err != nil {
		return nil, nil, err
	}

	testItems.app = app

	tmpData, err := insertSampleData(ctx, store)
	if err != nil {
		return nil, nil, err
	}

	testItems.tmpData = tmpData

	v1 := app.Group("/api/v1")
	peer.Routes(v1, store)

	return bootCleanup(testItems), testItems, nil
}

func insertSampleData(ctx context.Context, store *peerstore.Store) (*TmpData, error) {
	var tmpData = new(TmpData)

	tmpPeer, _, err := store.Insert(ctx, "/ip4/100.78.175.63/tcp/10002/p2p/12D3KooWERrdEepSi8HPRNsfjj3Nd7XcxV9SJcHdovpPLyYUtuch")
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

		return testItems.store.Close()
	}
}
