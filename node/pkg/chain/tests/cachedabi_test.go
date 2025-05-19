package tests

import (
	"strings"
	"testing"

	"bisonai.com/miko/node/pkg/chain/utils"
	errorsentinel "bisonai.com/miko/node/pkg/error"
	"github.com/kaiachain/kaia/accounts/abi"
	"github.com/stretchr/testify/assert"
)

func TestAbiCache(t *testing.T) {
	functionSignature := "testFunctionSignature"
	functionName := "testFunctionName"
	abiJSON := `[
		{
			"constant": true,
			"inputs": [],
			"name": "testFunctionName",
			"outputs": [{"name": "", "type": "uint256"}],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		}
	]`
	parsedAbi, err := abi.JSON(strings.NewReader(abiJSON))
	assert.NoError(t, err)

	// Test SetAbi
	utils.SetAbi(functionSignature, &parsedAbi, functionName)

	// Test GetAbi
	retrievedAbi, retrievedFunctionName, err := utils.GetAbi(functionSignature)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedAbi)
	assert.Equal(t, functionName, retrievedFunctionName)

	// Ensure the retrieved ABI matches the set ABI
	assert.Equal(t, parsedAbi.Methods[functionName].Name, retrievedAbi.Methods[functionName].Name)

	// Test GetAbi for non-existent entry
	_, _, err = utils.GetAbi("nonExistentSignature")
	assert.ErrorIs(t, err, errorsentinel.ErrChainCachedAbiNotFound)
}
