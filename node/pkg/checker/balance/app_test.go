//nolint:all
package balance

import (
	"context"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/testutil/testenv"
	"github.com/kaiachain/kaia/common"
	"github.com/stretchr/testify/assert"
)

const (
	testAddressWithFixedKlay = "0x2138824ef8741add09E8680F968e1d5D0AC155E0"
)

func TestGetBalance(t *testing.T) {
	// Reads a fixed-balance account on a live Kaia RPC endpoint.
	testenv.Env(t, "JSON_RPC_URL")
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
