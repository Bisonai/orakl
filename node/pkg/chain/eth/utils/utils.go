package utils

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	chain_common "bisonai.com/orakl/node/pkg/chain/common"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rs/zerolog/log"
)

func ReadContract(ctx context.Context, client *ethclient.Client, functionString string, contractAddress string, args ...interface{}) (interface{}, error) {
	functionName, inputs, outputs, err := chain_common.ParseMethodSignature(functionString)
	if err != nil {
		return nil, err
	}

	abi, err := GenerateViewABI(functionName, inputs, outputs)
	if err != nil {
		return nil, err
	}

	contractAddressHex := common.HexToAddress(contractAddress)
	callData, err := abi.Pack(functionName, args...)
	if err != nil {
		return nil, err
	}

	result, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &contractAddressHex,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, err
	}

	output, err := abi.Unpack(functionName, result)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func MakeDirectTx(ctx context.Context, client *ethclient.Client, contractAddressHex string, reporter string, functionString string, chainID *big.Int, args ...interface{}) (*types.Transaction, error) {
	functionName, inputs, outputs, err := chain_common.ParseMethodSignature(functionString)
	if err != nil {
		return nil, err
	}

	abi, err := GenerateCallABI(functionName, inputs, outputs)
	if err != nil {
		return nil, err
	}

	packed, err := abi.Pack(functionName, args...)
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(reporter)
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	contractAddress := common.HexToAddress(contractAddressHex)

	estimatedGas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		To:   &contractAddress,
		Data: packed,
	})
	if err != nil {
		log.Debug().Msg("failed to estimate gas, using default gas limit")
		estimatedGas = DEFAULT_GAS_LIMIT
	}

	if estimatedGas < DEFAULT_GAS_LIMIT {
		estimatedGas = DEFAULT_GAS_LIMIT
	}

	tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), estimatedGas, gasPrice, packed)
	return types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
}

func GenerateViewABI(functionName string, inputs string, outputs string) (*abi.ABI, error) {
	return generateABI(functionName, inputs, outputs, "view", false)
}

func GenerateCallABI(functionName string, inputs string, outputs string) (*abi.ABI, error) {
	return generateABI(functionName, inputs, outputs, "nonpayable", false)
}

func generateABI(functionName string, inputs string, outputs string, stateMutability string, payable bool) (*abi.ABI, error) {
	inputArgs := chain_common.MakeAbiFuncAttribute(inputs)
	outputArgs := chain_common.MakeAbiFuncAttribute(outputs)

	json := fmt.Sprintf(`[
		{
			"constant": false,
			"inputs": [%s],
			"name": "%s",
			"outputs": [%s],
			"payable": %t,
			"stateMutability": "%s",
			"type": "function"
		}
	]
	`, inputArgs, functionName, outputArgs, payable, stateMutability)

	parsedABI, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		return nil, err
	}

	return &parsedABI, nil
}

func GetChainID(ctx context.Context, client *ethclient.Client) (*big.Int, error) {
	return client.NetworkID(ctx)
}

func SubmitRawTx(ctx context.Context, client *ethclient.Client, tx *types.Transaction) error {
	err := client.SendTransaction(ctx, tx)
	if err != nil {
		return err
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return err
	}

	if receipt.Status != 1 {
		return errors.New("tx failed")
	}

	log.Debug().Any("tx", tx.Hash().Hex()).Msg("tx submitted successfully")
	return nil
}

func SubmitRawTxString(ctx context.Context, client *ethclient.Client, rawTx string) error {
	rawTxBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		return err
	}

	tx := new(types.Transaction)
	err = rlp.DecodeBytes(rawTxBytes, tx)
	if err != nil {
		return err
	}

	return SubmitRawTx(ctx, client, tx)
}

func GetWallets(ctx context.Context) ([]string, error) {
	// TODO: update db structure to save separate wallets each chain
	return []string{}, nil
}
