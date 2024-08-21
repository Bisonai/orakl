//nolint:all
package balance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/klaytn/klaytn/common"
	"github.com/stretchr/testify/assert"
)

const (
	testAddr0 = "0x5aD1dc8e413c2c3364294d784aE8c9FafD43f079"
	testAddr1 = "0x1AD018aa154cA85E98A49Ba04344212350A8754b"

	testAddressWithFixedKlay = "0x2138824ef8741add09E8680F968e1d5D0AC155E0"
)

func TestLoadWalletFromMikoApi(t *testing.T) {
	ctx := context.Background()
	mockServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`[{"pk":"abc", "address":"` + testAddr0 + `", "service":"REQUEST_RESPONSE"},{"pk":"def", "address":"` + testAddr1 + `", "service":"VRF"}]`))
	}))
	defer mockServer.Close()

	wallets, err := loadWalletFromMikoApi(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.Equal(t, 2, len(wallets))
	assert.Equal(t, testAddr0, wallets[0].Address.Hex())
	assert.Equal(t, testAddr1, wallets[1].Address.Hex())
}

func TestLoadWalletFromPor(t *testing.T) {
	ctx := context.Background()
	mockServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(testAddr0))
	}))
	defer mockServer.Close()

	wallet, err := loadWalletFromPor(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.Equal(t, testAddr0, wallet.Address.Hex())
}

func TestLoadWalletFromDelegator(t *testing.T) {
	ctx := context.Background()
	mockServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`"` + testAddr0 + `"`))
	}))
	defer mockServer.Close()

	wallet, err := loadWalletFromDelegator(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.Equal(t, testAddr0, wallet.Address.Hex())
}

func TestGetBalance(t *testing.T) {
	ctx := context.Background()
	err := setClient(os.Getenv("JSON_RPC_URL"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	balance, err := getBalance(ctx, common.HexToAddress(testAddressWithFixedKlay))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	assert.Equal(t, float64(50), balance)
}
