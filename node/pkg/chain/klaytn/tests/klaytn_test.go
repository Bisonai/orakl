package tests

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	chain_common "bisonai.com/orakl/node/pkg/chain/common"
	"bisonai.com/orakl/node/pkg/chain/klaytn/helper"
	klaytn_utils "bisonai.com/orakl/node/pkg/chain/klaytn/utils"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/stretchr/testify/assert"
)

var (
	ErrEmptyContractAddress = errors.New("contract address is empty")
	ErrEmptyFunctionString  = errors.New("function string is empty")
)

func TestNewKlaytnHelper(t *testing.T) {
	ctx := context.Background()
	txHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	txHelper.Close()
}

func TestNextReporter(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	reporter := klaytnHelper.NextReporter()
	if reporter == "" {
		t.Errorf("Unexpected reporter: %v", reporter)
	}
}

func TestMakeDirectTx(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	tests := []struct {
		name            string
		contractAddress string
		functionString  string
		expectedError   error
	}{
		{
			name:            "Test case 1",
			contractAddress: "0x93120927379723583c7a0dd2236fcb255e96949f",
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
			contractAddress: "0x93120927379723583c7a0dd2236fcb255e96949f",
			functionString:  "",
			expectedError:   ErrEmptyFunctionString,
		},
	}

	for _, test := range tests {
		directTx, err := klaytnHelper.MakeDirectTx(ctx, test.contractAddress, test.functionString)
		if err != nil {
			if err.Error() != test.expectedError.Error() {
				t.Errorf("Test case %s: Expected error '%v', but got '%v'", test.name, test.expectedError, err)
			}
		}
		if err == nil {
			assert.Equal(t, strings.ToLower(directTx.To().Hex()), test.contractAddress)
			assert.Equal(t, directTx.Value().Cmp(big.NewInt(0)), 0)
		}
	}
}

func TestMakeFeeDelegatedTx(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	tests := []struct {
		name            string
		contractAddress string
		functionString  string
		expectedError   error
	}{
		{
			name:            "Test case 1",
			contractAddress: "0x93120927379723583c7a0dd2236fcb255e96949f",
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
			contractAddress: "0x93120927379723583c7a0dd2236fcb255e96949f",
			functionString:  "",
			expectedError:   ErrEmptyFunctionString,
		},
	}

	for _, test := range tests {
		feeDelegatedTx, err := klaytnHelper.MakeFeeDelegatedTx(ctx, test.contractAddress, test.functionString)
		if err != nil {
			if err.Error() != test.expectedError.Error() {
				t.Errorf("Test case %s: Expected error '%v', but got '%v'", test.name, test.expectedError, err)
			}
		}
		if err == nil {
			assert.Equal(t, strings.ToLower(feeDelegatedTx.To().Hex()), test.contractAddress)
			assert.Equal(t, feeDelegatedTx.Value().Cmp(big.NewInt(0)), 0)
			assert.Equal(t, feeDelegatedTx.Type(), types.TxTypeFeeDelegatedSmartContractExecution)
		}
	}
}

func TestTxToHashToTx(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	rawTx, err := klaytnHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hash := klaytn_utils.TxToHash(rawTx)
	assert.NotEqual(t, hash, "")

	tx, err := klaytn_utils.HashToTx(hash)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, tx.Equal(rawTx), true)
}

func TestGenerateABI(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	functionName, inputs, outputs, err := chain_common.ParseMethodSignature("increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	abi, err := klaytn_utils.GenerateCallABI(functionName, inputs, outputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, "increment", abi.Methods["increment"].Name)

	assert.NotEqual(t, abi, nil)
}

func TestGenerateViewABI(t *testing.T) {
	functionName, inputs, outputs, err := chain_common.ParseMethodSignature("function slot0() external view returns (uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint8 feeProtocol, bool unlocked)")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	abi, err := klaytn_utils.GenerateViewABI(functionName, inputs, outputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, "slot0", abi.Methods["slot0"].Name)

	assert.NotEqual(t, abi, nil)
}

func TestSubmitRawTxString(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	rawTx, err := klaytnHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	signedTx, err := klaytnHelper.SignTxByFeePayer(ctx, rawTx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rawTxString := klaytn_utils.TxToHash(signedTx)
	err = klaytnHelper.SubmitRawTxString(ctx, rawTxString)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestReadContract(t *testing.T) {
	ctx := context.Background()
	klaytnHelper, err := helper.NewKlaytnHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer klaytnHelper.Close()

	result, err := klaytnHelper.ReadContract(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "COUNTER() returns (uint256)")

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

func TestGenerateViewABIWithInvalidSignature(t *testing.T) {
	functionName, inputs, outputs, err := chain_common.ParseMethodSignature("invalidFunctionSignature")
	if err == nil {
		t.Errorf("Expected an error for invalid function signature, got nil")
	}

	_, err = klaytn_utils.GenerateViewABI(functionName, inputs, outputs)
	if err == nil {
		t.Errorf("Expected an error when generating ABI with invalid function signature, got nil")
	}
}

func TestGenerateViewABIWithEmptySignature(t *testing.T) {
	functionName, inputs, outputs, err := chain_common.ParseMethodSignature("")
	if err == nil {
		t.Errorf("Expected an error for empty function signature, got nil")
	}

	_, err = klaytn_utils.GenerateViewABI(functionName, inputs, outputs)
	if err == nil {
		t.Errorf("Expected an error when generating ABI with empty function signature, got nil")
	}
}
