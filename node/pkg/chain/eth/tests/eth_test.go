package tests

import (
	"context"
	"errors"
	"math/big"
	"testing"

	chain_common "bisonai.com/orakl/node/pkg/chain/common"
	"bisonai.com/orakl/node/pkg/chain/eth/helper"
	eth_utils "bisonai.com/orakl/node/pkg/chain/eth/utils"
	"github.com/stretchr/testify/assert"
)

var (
	ErrEmptyContractAddress = errors.New("contract address is empty")
	ErrEmptyFunctionString  = errors.New("function string is empty")
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

	tests := []struct {
		name            string
		contractAddress string
		functionString  string
		expectedError   error
	}{
		{
			name:            "Test case 1",
			contractAddress: "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b",
			functionString:  "increment()",
			expectedError:   nil,
		},
		{
			name:            "Test case 2",
			contractAddress: "",
			functionString:  "increment()",
			expectedError:   ErrEmptyContractAddress,
		},
		{
			name:            "Test case 3",
			contractAddress: "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b",
			functionString:  "",
			expectedError:   ErrEmptyFunctionString,
		},
	}

	for _, test := range tests {
		directTx, err := ethHelper.MakeDirectTx(ctx, test.contractAddress, test.functionString)
		if err != nil && err.Error() != test.expectedError.Error() {
			t.Errorf("Test case %s: Expected error '%v', but got '%v'", test.name, test.expectedError, err)
		}
		if directTx != nil {
			assert.NotEqual(t, directTx, nil)
			assert.Equal(t, directTx.To().Hex(), test.contractAddress)
		}
	}
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
