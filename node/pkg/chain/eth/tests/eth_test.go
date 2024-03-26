package tests

import (
	"context"
	"math/big"
	"testing"

	chain_common "bisonai.com/orakl/node/pkg/chain/common"
	"bisonai.com/orakl/node/pkg/chain/eth/helper"
	eth_utils "bisonai.com/orakl/node/pkg/chain/eth/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewEthHelper(t *testing.T) {
	ctx := context.Background()
	ethHelper, err := helper.NewEthHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	ethHelper.Close()
}

func TestNextReporter(t *testing.T) {
	ctx := context.Background()
	ethHelper, err := helper.NewEthHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer ethHelper.Close()

	reporter := ethHelper.NextReporter()
	if reporter == "" {
		t.Errorf("Unexpected reporter: %v", reporter)
	}
}

func TestMakeDirectTx(t *testing.T) {
	ctx := context.Background()
	ethHelper, err := helper.NewEthHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer ethHelper.Close()

	directTx, err := ethHelper.MakeDirectTx(ctx, "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, directTx.To().Hex(), "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b")
	assert.Equal(t, directTx.Value().Cmp(big.NewInt(0)), 0)
}

func TestGenerateABI(t *testing.T) {
	ctx := context.Background()
	ethHelper, err := helper.NewEthHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer ethHelper.Close()

	functionName, inputs, outputs, err := chain_common.ParseMethodSignature("increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	abi, err := eth_utils.GenerateCallABI(functionName, inputs, outputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, "increment", abi.Methods["increment"].Name)
	assert.NotEqual(t, abi, nil)
}

func TestReadContract(t *testing.T) {
	ctx := context.Background()
	ethHelper, err := helper.NewEthHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer ethHelper.Close()

	result, err := ethHelper.ReadContract(ctx, "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b", "function count() external view returns (uint256)")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotEqual(t, result, nil)
	if resultSlice, ok := result.([]interface{}); ok {
		if len(resultSlice) > 0 {
			if bigIntResult, ok := resultSlice[0].(*big.Int); ok {
				assert.NotEqual(t, bigIntResult, nil)
			} else {
				t.Errorf("Unexpected error: %v", err)
			}
		} else {
			t.Errorf("Unexpected error: %v", "result is empty")
		}
	}
}

func TestSubmitRawTxString(t *testing.T) {
	t.Skip("Skipping test, need wallet with enough testnet ETH")
	ctx := context.Background()
	ethHelper, err := helper.NewEthHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer ethHelper.Close()

	rawTx, err := ethHelper.MakeDirectTx(ctx, "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = ethHelper.SubmitRawTx(ctx, rawTx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
