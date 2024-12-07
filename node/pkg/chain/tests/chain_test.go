package tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	"bisonai.com/miko/node/pkg/chain/noncemanager"
	"bisonai.com/miko/node/pkg/chain/utils"
	"bisonai.com/miko/node/pkg/db"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	InsertProviderUrlQuery = "INSERT INTO provider_urls (chain_id, url, priority) VALUES (@chain_id, @url, @priority)"
	maxTxSubmissionRetries = 3
)

func TestNewKaiaHelper(t *testing.T) {
	ctx := context.Background()
	err := db.QueryWithoutResult(ctx, InsertProviderUrlQuery, map[string]any{
		"chain_id": 1001,
		"url":      "https://public-en-kairos.node.kaia.io",
		"priority": 2,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	kaiaHelper.Close()
	err = db.QueryWithoutResult(ctx, "DELETE FROM provider_urls;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewSigner(t *testing.T) {
	ctx := context.Background()

	_, err := helper.NewSigner(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNewEthHelper(t *testing.T) {
	ctx := context.Background()

	err := db.QueryWithoutResult(ctx, InsertProviderUrlQuery, map[string]any{
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
	ethHelper.Close()
	err = db.QueryWithoutResult(ctx, "DELETE FROM provider_urls;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMakeDirectTx(t *testing.T) {
	ctx := context.Background()

	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	t.Log("helper initialized")

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
		t.Log(test.name)
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

func TestSubmitDelegeted(t *testing.T) {
	t.Skip()
	ctx := context.Background()

	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	err = kaiaHelper.SubmitDelegatedFallbackDirect(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSubmitDelegetedFallbackDirectConcurrent(t *testing.T) {
	ctx := context.Background()
	noncemanager.ResetInstance()
	kaiaHelper, err := helper.NewChainHelper(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer kaiaHelper.Close()

	const numCalls = 3

	var wg sync.WaitGroup
	wg.Add(numCalls)

	errCh := make(chan error, numCalls)

	submitTx := func() {
		defer wg.Done()
		err := kaiaHelper.SubmitDelegatedFallbackDirect(ctx, "0x93120927379723583c7a0dd2236fcb255e96949f", "increment()")
		errCh <- err
	}

	for i := 0; i < numCalls; i++ {
		go submitTx()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
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

func TestMakeGlobalAggregateProof(t *testing.T) {
	ctx := context.Background()
	s, err := helper.NewSigner(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	timestamp := time.Now()

	proof, err := s.MakeGlobalAggregateProof(200000000, timestamp, "test-aggregate")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.NotEqual(t, proof, nil)

	hash := utils.Value2HashForSign(200000000, timestamp.UnixMilli(), "test-aggregate")
	addr, err := utils.RecoverSigner(hash, proof)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	pk, err := utils.StringToPk(os.Getenv("SIGNER_PK"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	addrFromEnv := crypto.PubkeyToAddress(pk.PublicKey)

	assert.Equal(t, addrFromEnv.Hex(), addr.Hex())
}

func TestMakeMultiGlobalAggregateProof(t *testing.T) {
	ctx := context.Background()

	pubKeys := []string{"0x75EC9060d2C3260c0009297ca093320A98B8741a", "0xbC1259FB2AaD1881Dff9317e22722bffa4492543"}
	privKeys := []string{"0x27894b84849f129e08f37634be4e8ccc4c7267d824eb8cfd285185854ba5b78d", "0xc4aeea4b48e0cba6651c33ef86b96c1ae8c9d0229fd4605d6404fc8f8b6b180b"}

	timestamp := time.Now()
	value := int64(200000000)

	signHelpers := make([]*helper.Signer, 0, len(pubKeys))
	for _, pk := range privKeys {
		s, err := helper.NewSigner(ctx, helper.WithSignerPk(pk))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		signHelpers = append(signHelpers, s)
	}

	rawProofs := [][]byte{}

	for _, s := range signHelpers {
		proof, err := s.MakeGlobalAggregateProof(value, timestamp, "test-aggregate")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		rawProofs = append(rawProofs, proof)
		assert.NotNil(t, proof)
	}

	hash := utils.Value2HashForSign(value, timestamp.UnixMilli(), "test-aggregate")

	merged := bytes.Join(rawProofs, nil)
	proofs := make([][]byte, 0, len(merged)/65)
	for i := 0; i < len(merged); i += 65 {
		proofs = append(proofs, merged[i:i+65])
	}

	signers := []common.Address{}
	for _, p := range proofs {
		signer, err := utils.RecoverSigner(hash, p)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		signers = append(signers, signer)
	}

	for i, signer := range signers {
		assert.Equal(t, pubKeys[i], signer.Hex())
	}
}

func TestGlobalAggregateProofMergeAndSplit(t *testing.T) {
	ctx := context.Background()
	s, err := helper.NewSigner(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	timestamp := time.Now()

	sampleValues := []int64{100000000, 200000000, 300000000}
	sampleProofs := [][]byte{}

	for _, v := range sampleValues {
		proof, err := s.MakeGlobalAggregateProof(v, timestamp, "test-aggregate")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		sampleProofs = append(sampleProofs, proof)
	}

	merged := bytes.Join(sampleProofs, nil)

	proofs := make([][]byte, 0, len(merged)/65)
	for i := 0; i < len(merged); i += 65 {
		proofs = append(proofs, merged[i:i+65])
	}

	for i, p := range proofs {
		if !bytes.Equal(p, sampleProofs[i]) {
			t.Errorf("Test case %d: Expected proof %x, but got %x", i, sampleProofs[i], p)
		}
	}

	hashes := [][]byte{}
	for _, v := range sampleValues {
		hash := utils.Value2HashForSign(v, timestamp.UnixMilli(), "test-aggregate")
		hashes = append(hashes, hash)
	}

	for i, h := range hashes {
		addr, err := utils.RecoverSigner(h, proofs[i])
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		pk, err := utils.StringToPk(os.Getenv("SIGNER_PK"))
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		addrFromEnv := crypto.PubkeyToAddress(pk.PublicKey)

		assert.Equal(t, addrFromEnv.Hex(), addr.Hex())
	}
}

func TestNewPk(t *testing.T) {
	pk, pkHex, err := utils.NewPk(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.NotEqual(t, nil, pk)
	assert.NotEqual(t, "", pkHex)

	addr, err := utils.StringPkToAddressHex(pkHex)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.NotEqual(t, nil, addr)
}

func TestSignerTableSingleEntry(t *testing.T) {
	ctx := context.Background()

	//cleanup
	err := db.QueryWithoutResult(ctx, "DELETE FROM signer;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mockPk1 := "0xf558f6ef079e7fe096eac4497c1e07af6a867f86411c42aad3757f7768316ceb"
	mockPk2 := "0x81abf286f673fc51d2b0d6811f760665893838bdc41fa3caca0dd8d83e0ff105"

	result, err := db.QueryRow[utils.Wallet](ctx, "SELECT id, pk FROM signer LIMIT 1;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, int64(0), result.ID)
	assert.Equal(t, "", result.PK)

	err = utils.StoreSignerPk(ctx, mockPk1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	loadedPk, err := utils.LoadSignerPk(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, strings.TrimPrefix(mockPk1, "0x"), strings.TrimPrefix(loadedPk, "0x"))

	err = utils.StoreSignerPk(ctx, mockPk2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rowsResult, err := db.QueryRows[utils.Wallet](ctx, "SELECT id, pk FROM signer;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, 1, len(rowsResult))

	loadedPk, err = utils.LoadSignerPk(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	assert.Equal(t, strings.TrimPrefix(mockPk2, "0x"), strings.TrimPrefix(loadedPk, "0x"))

	//cleanup
	err = db.QueryWithoutResult(ctx, "DELETE FROM signer;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSignerRenew(t *testing.T) {
	ctx := context.Background()

	contractAddr := os.Getenv("SUBMISSION_PROXY_CONTRACT")
	if contractAddr == "" {
		t.Skip("Skipping test because SUBMISSION_PROXY_CONTRACT is not set")
	}

	s, err := helper.NewSigner(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expiration, err := s.LoadExpiration(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.False(t, expiration.IsZero())
	fmt.Println(expiration)

	renewalRequired := s.IsRenewalRequired()
	assert.False(t, renewalRequired)

	oldPK := s.PK
	oldPKBytes := crypto.FromECDSA(oldPK)
	oldPKHex := hex.EncodeToString(oldPKBytes)
	oldSignerAddr, err := utils.StringPkToAddressHex(oldPKHex)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	newPK, newPKHex, err := utils.NewPk(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	newSignerAddr, err := utils.StringPkToAddressHex(newPKHex)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = s.Renew(ctx, newPK, newPKHex)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	newExpiration, err := s.LoadExpiration(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.False(t, newExpiration.IsZero())
	assert.Greater(t, newExpiration.Unix(), expiration.Unix())

	//cleanup
	chainHelperForCleanup, err := helper.NewChainHelper(ctx, helper.WithReporterPk(oldPKHex))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	addOracleFunctionSignature := "addOracle(address _oracle) external returns (uint256)"
	removeOracleFunctionSignature := "function removeOracle(address _oracle) external"

	err = chainHelperForCleanup.SubmitDelegatedFallbackDirect(ctx, contractAddr, addOracleFunctionSignature, maxTxSubmissionRetries, common.HexToAddress(oldSignerAddr))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = chainHelperForCleanup.SubmitDelegatedFallbackDirect(ctx, contractAddr, removeOracleFunctionSignature, maxTxSubmissionRetries, common.HexToAddress(newSignerAddr))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = db.QueryWithoutResult(ctx, "DELETE FROM signer;", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
