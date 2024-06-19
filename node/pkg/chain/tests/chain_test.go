package tests

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	"bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	InsertProviderUrlQuery = "INSERT INTO provider_urls (chain_id, url, priority) VALUES (@chain_id, @url, @priority)"
)

func TestNewKaiaHelper(t *testing.T) {
	ctx := context.Background()

	err := db.QueryWithoutResult(ctx, InsertProviderUrlQuery, map[string]any{
		"chain_id": 1001,
		"url":      "https://public-en.kairos.node.kaia.io",
		"priority": 1,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)

	}
	err = db.QueryWithoutResult(ctx, InsertProviderUrlQuery, map[string]any{
		"chain_id": 1001,
		"url":      "https://public-en.kairos.node.kaia.io",
		"priority": 2,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, 3, kaiaHelper.NumClients())

	kaiaHelper.Close()
	err = db.QueryWithoutResult(ctx, "DELETE FROM provider_urls;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewChainHelper(t *testing.T) {
	_, err := helper.NewSignHelper("")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewEthHelper(t *testing.T) {
	ctx := context.Background()

	err := db.QueryWithoutResult(ctx, InsertProviderUrlQuery, map[string]any{
		"chain_id": 11155111,
		"url":      "https://sepolia.gateway.tenderly.co",
		"priority": 1,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	err = db.QueryWithoutResult(ctx, InsertProviderUrlQuery, map[string]any{
		"chain_id": 11155111,
		"url":      "wss://ethereum-sepolia-rpc.publicnode.com",
		"priority": 2,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	ethHelper, err := helper.NewChainHelper(ctx, helper.WithBlockchainType(helper.Ethereum))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, 3, ethHelper.NumClients())
	ethHelper.Close()
	err = db.QueryWithoutResult(ctx, "DELETE FROM provider_urls;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNextReporter(t *testing.T) {
	ctx := context.Background()
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	reporter := kaiaHelper.NextReporter()
	if reporter == "" {
		t.Errorf("Unexpected reporter: %v", reporter)
	}
}

func TestMakeDirectTx(t *testing.T) {
	ctx := context.Background()
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

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
			expectedError:   errorSentinel.ErrChainEmptyAddressParam,
		},
		{
			name:            "Test case 3",
			contractAddress: "0x93120927379723583c7a0dd2236fcb255e96949f",
			functionString:  "",
			expectedError:   errorSentinel.ErrChainEmptyFuncStringParam,
		},
	}

	for _, test := range tests {
		directTx, err := kaiaHelper.MakeDirectTx(ctx, test.contractAddress, test.functionString)
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
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

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
			expectedError:   errorSentinel.ErrChainEmptyAddressParam,
		},
		{
			name:            "Test case 3",
			contractAddress: "0x93120927379723583c7a0dd2236fcb255e96949f",
			functionString:  "",
			expectedError:   errorSentinel.ErrChainEmptyFuncStringParam,
		},
	}

	for _, test := range tests {
		feeDelegatedTx, err := kaiaHelper.MakeFeeDelegatedTx(ctx, test.contractAddress, test.functionString, 0)
		if err != nil {
			assert.ErrorIs(t, err, test.expectedError)
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
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	rawTx, err := kaiaHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()", 0)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	hash := utils.TxToHash(rawTx)
	assert.NotEqual(t, hash, "")

	tx, err := utils.HashToTx(hash)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, tx.Equal(rawTx), true)
}

func TestGenerateCallABI(t *testing.T) {
	functionName, inputs, outputs, err := utils.ParseMethodSignature("increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	abi, err := utils.GenerateCallABI(functionName, inputs, outputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, "increment", abi.Methods["increment"].Name)

	assert.NotEqual(t, abi, nil)
}

func TestGenerateViewABI(t *testing.T) {
	functionName, inputs, outputs, err := utils.ParseMethodSignature("function slot0() external view returns (uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint8 feeProtocol, bool unlocked)")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	abi, err := utils.GenerateViewABI(functionName, inputs, outputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, "slot0", abi.Methods["slot0"].Name)

	assert.NotEqual(t, abi, nil)
}

func TestSubmitRawTxString(t *testing.T) {
	// testing based on baobab testnet
	ctx := context.Background()
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	rawTx, err := kaiaHelper.MakeFeeDelegatedTx(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()", 0)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	signedTx, err := kaiaHelper.SignTxByFeePayer(ctx, rawTx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rawTxString := utils.TxToHash(signedTx)
	err = kaiaHelper.SubmitRawTxString(ctx, rawTxString)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestReadContract(t *testing.T) {
	// testing based on baobab testnet
	ctx := context.Background()
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	result, err := kaiaHelper.ReadContract(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "COUNTER() returns (uint256)")

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

func TestReadContractWithEthHelper(t *testing.T) {
	// testing based on sepolia eth testnet
	ctx := context.Background()
	ethHelper, err := helper.NewChainHelper(ctx, helper.WithBlockchainType(helper.Ethereum))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	result, err := ethHelper.ReadContract(ctx, "0x72C8f1933A0C0a9ad53D6CdDAF2e1Ce2F6075D2b", "count() returns (uint256)")
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
	functionName, inputs, outputs, err := utils.ParseMethodSignature("invalidFunctionSignature")
	if err == nil {
		t.Errorf("Expected an error for invalid function signature, got nil")
	}

	_, err = utils.GenerateViewABI(functionName, inputs, outputs)
	if err == nil {
		t.Errorf("Expected an error when generating ABI with invalid function signature, got nil")
	}
}

func TestGenerateViewABIWithEmptySignature(t *testing.T) {
	functionName, inputs, outputs, err := utils.ParseMethodSignature("")
	if err == nil {
		t.Errorf("Expected an error for empty function signature, got nil")
	}

	_, err = utils.GenerateViewABI(functionName, inputs, outputs)
	if err == nil {
		t.Errorf("Expected an error when generating ABI with empty function signature, got nil")
	}
}

func TestParseMethodSignature(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedName   string
		expectedInput  string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "Test case 1",
			input:          "function foo()",
			expectedName:   "foo",
			expectedInput:  "",
			expectedOutput: "",
			expectedError:  nil,
		},
		{
			name:           "Test case 2",
			input:          "function bar(arg1 int, arg2 string) returns (bool)",
			expectedName:   "bar",
			expectedInput:  "arg1 int, arg2 string",
			expectedOutput: "bool",
			expectedError:  nil,
		},
		{
			name:           "Test case 3",
			input:          "function baz(arg1 int, arg2 string) returns (bool, string)",
			expectedName:   "baz",
			expectedInput:  "arg1 int, arg2 string",
			expectedOutput: "bool, string",
			expectedError:  nil,
		},
		{
			name:           "Test case 4",
			input:          "",
			expectedName:   "",
			expectedInput:  "",
			expectedOutput: "",
			expectedError:  errorSentinel.ErrChainEmptyNameParam,
		},
	}

	for _, test := range tests {
		funcName, inputArgs, outputArgs, err := utils.ParseMethodSignature(test.input)
		if err != nil {
			assert.ErrorIs(t, err, test.expectedError)
		}
		if funcName != test.expectedName {
			t.Errorf("Test case %s: Expected function name '%s', but got '%s'", test.name, test.expectedName, funcName)
		}
		if inputArgs != test.expectedInput {
			t.Errorf("Test case %s: Expected input arguments '%s', but got '%s'", test.name, test.expectedInput, inputArgs)
		}
		if outputArgs != test.expectedOutput {
			t.Errorf("Test case %s: Expected output arguments '%s', but got '%s'", test.name, test.expectedOutput, outputArgs)
		}
	}
}

func TestMakeAbiFuncAttribute(t *testing.T) {
	tests := []struct {
		name     string
		args     string
		expected string
	}{
		{
			name:     "Test case 1",
			args:     "",
			expected: "",
		},
		{
			name:     "Test case 2",
			args:     "int",
			expected: `{"type":"int"}`,
		},
		{
			name:     "Test case 3",
			args:     "string",
			expected: `{"type":"string"}`,
		},
		{
			name: "Test case 4",
			args: "int, string",
			expected: `{"type":"int"},
{"type":"string"}`,
		},
		{
			name: "Test case 5",
			args: "int arg1, string arg2",
			expected: `{"type":"int","name":"arg1"},
{"type":"string","name":"arg2"}`,
		},
		{
			name: "Test case 6",
			args: "address[] memory addresses, uint256[] memory amounts",
			expected: `{"type":"address[]","name":"addresses"},
{"type":"uint256[]","name":"amounts"}`,
		},
	}

	for _, test := range tests {
		result := utils.MakeAbiFuncAttribute(test.args)
		if result != test.expected {
			t.Errorf("Test case %s: Expected '%s', but got '%s'", test.name, test.expected, result)
		}
	}
}

func TestGetWallets(t *testing.T) {
	ctx := context.Background()
	testPk := "cbebf778dd0a62952e6caa9d51eefc6ec9242c1111e7b7e1165485a2041cab2b"

	err := utils.InsertWallet(ctx, testPk)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer func() {
		err = db.QueryWithoutResult(ctx, "DELETE FROM wallets;", nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}()

	wallets, err := utils.GetWallets(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Contains(t, wallets, testPk)
}

func TestInsertWalletEmptyDbEnv(t *testing.T) {
	ctx := context.Background()
	err := os.Setenv("DATABASE_URL", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	testPk := "cbebf778dd0a62952e6caa9d51eefc6ec9242c1111e7b7e1165485a2041cab2b"

	err = utils.InsertWallet(ctx, testPk)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	wallets, err := utils.GetWallets(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotContains(t, wallets, testPk)
}

func TestMakeGlobalAggregateProof(t *testing.T) {
	s, err := helper.NewSignHelper("")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	timestamp := time.Now()

	proof, err := s.MakeGlobalAggregateProof(200000000, timestamp, "test-aggregate")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotEqual(t, proof, nil)

	hash := utils.Value2HashForSign(200000000, timestamp.Unix(), "test-aggregate")
	addr, err := utils.RecoverSigner(hash, proof)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	pk, err := utils.StringToPk(os.Getenv("KAIA_REPORTER_PK"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	addrFromEnv := crypto.PubkeyToAddress(pk.PublicKey)

	assert.Equal(t, addrFromEnv.Hex(), addr.Hex())
}
