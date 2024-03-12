//nolint:all

package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/wallet"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestWalletInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockWallet := wallet.WalletInsertModel{
		Pk: "test_pk_v2",
	}

	readResultBefore, err := GetRequest[[]wallet.WalletModel](testItems.app, "/api/v1/wallet", nil)
	if err != nil {
		t.Fatalf("error getting wallets before: %v", err)
	}

	insertResult, err := PostRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet", mockWallet)
	if err != nil {
		t.Fatalf("error inserting wallet: %v", err)
	}

	assert.Equal(t, insertResult.Pk, mockWallet.Pk)

	readResultAfter, err := GetRequest[[]wallet.WalletModel](testItems.app, "/api/v1/wallet", nil)
	if err != nil {
		t.Fatalf("error getting wallets after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more wallets after insertion")

	err = db.QueryWithoutResult(context.Background(), wallet.DeleteWalletById, map[string]any{"id": insertResult.Id})
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestWalletGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]wallet.WalletModel](testItems.app, "/api/v1/wallet", nil)
	if err != nil {
		t.Fatalf("error getting wallets: %v", err)
	}

	assert.Greaterf(t, len(readResult), 0, "expected to have at least one wallet")
}

func TestWalletGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultById, err := GetRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.FormatInt(*testItems.tmpData.wallet.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting wallet by id: %v", err)
	}

	assert.Equal(t, testItems.tmpData.wallet.Pk, readResultById.Pk)
}

func TestWalletUpdateById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockWallet := wallet.WalletInsertModel{
		Pk: "test_pk_v2",
	}

	updateResult, err := PatchRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.FormatInt(*testItems.tmpData.wallet.Id, 10), map[string]any{"pk": mockWallet.Pk})
	if err != nil {
		t.Fatalf("error updating wallet: %v", err)
	}

	assert.Equal(t, updateResult.Pk, mockWallet.Pk)
}

func TestWalletDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]wallet.WalletModel](testItems.app, "/api/v1/wallet", nil)
	if err != nil {
		t.Fatalf("error getting wallets before: %v", err)
	}

	err = db.QueryWithoutResult(context.Background(), wallet.DeleteWalletById, map[string]any{"id": testItems.tmpData.wallet.Id})
	if err != nil {
		t.Fatalf("error deleting wallet: %v", err)
	}

	readResultAfter, err := GetRequest[[]wallet.WalletModel](testItems.app, "/api/v1/wallet", nil)
	if err != nil {
		t.Fatalf("error getting wallets after: %v", err)
	}

	assert.Lessf(t, len(readResultAfter), len(readResultBefore), "expected to have less wallets after deletion")
}
