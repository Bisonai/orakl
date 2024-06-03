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
		Pk: "0x7b48c1fd1861ebc850e3a8629198e9c4d33fc16ff995162a25438b532c42253d",
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

	readSingle, err := GetRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.Itoa(int(*insertResult.ID)), nil)
	if err != nil {
		t.Fatalf("error getting wallet by id: %v", err)
	}
	assert.Equalf(t, readSingle.Pk, mockWallet.Pk, "expected to have the same wallet")

	err = db.QueryWithoutResult(context.Background(), wallet.DeleteWalletById, map[string]any{"id": insertResult.ID})
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
	assert.Equalf(t, readResult[0].Pk, testItems.tmpData.wallet.Pk, "expected to have the same wallet")
}

func TestWalletGetAddress(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]string](testItems.app, "/api/v1/wallet/addresses", nil)
	if err != nil {
		t.Fatalf("error getting addresses: %v", err)
	}

	assert.Greaterf(t, len(readResult), 0, "expected to have at least one address")
	assert.Equalf(t, readResult[0], "0xd45bd119bE9D4EE5dCd642978648142681caa7e6", "expected to have the same address")
}

func TestWalletGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultById, err := GetRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.Itoa(int(*testItems.tmpData.wallet.ID)), nil)
	if err != nil {
		t.Fatalf("error getting wallet by id: %v", err)
	}
	failReadByInvalidId, err := GetRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/0", nil)
	if err != nil {
		t.Fatalf("error getting wallet by invalid id: %v", err)
	}

	assert.Equal(t, failReadByInvalidId, wallet.WalletModel{})

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
		Pk: "0x7b48c1fd1861ebc850e3a8629198e9c4d33fc16ff995162a25438b532c42253d",
	}

	beforeUpdate, err := GetRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.Itoa(int(*testItems.tmpData.wallet.ID)), nil)
	if err != nil {
		t.Fatalf("error getting wallet by id before update: %v", err)
	}

	updateResult, err := PatchRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.Itoa(int(*testItems.tmpData.wallet.ID)), map[string]any{"pk": mockWallet.Pk})
	if err != nil {
		t.Fatalf("error updating wallet: %v", err)
	}

	assert.Equal(t, updateResult.Pk, mockWallet.Pk)
	assert.NotEqual(t, beforeUpdate.Pk, updateResult.Pk)
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

	removeResult, err := DeleteRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.Itoa(int(*testItems.tmpData.wallet.ID)), nil)
	if err != nil {
		t.Fatalf("error deleting wallet: %v", err)
	}

	readResultAfter, err := GetRequest[[]wallet.WalletModel](testItems.app, "/api/v1/wallet", nil)
	if err != nil {
		t.Fatalf("error getting wallets after: %v", err)
	}

	assert.Lessf(t, len(readResultAfter), len(readResultBefore), "expected to have less wallets after deletion")

	failReadAfterDelete, err := GetRequest[wallet.WalletModel](testItems.app, "/api/v1/wallet/"+strconv.Itoa(int(*removeResult.ID)), nil)
	if err != nil {
		t.Fatalf("error getting wallet by id after deletion: %v", err)
	}
	assert.Equalf(t, failReadAfterDelete, wallet.WalletModel{}, "expected to have no wallet after deletion")
}
