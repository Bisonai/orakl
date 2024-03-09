package utils

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/klaytn/klaytn/accounts/abi"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/rlp"
)

func TestMakeRawTx(ctx context.Context) (string, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return "", err
	}

	privateKey, err := crypto.HexToECDSA(os.Getenv("REPORTER_PK"))
	if err != nil {
		return "", err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", err
	}

	value := big.NewInt(1000000000000000000)
	gasLimit := uint64(90000)
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	toAddress := common.HexToAddress("0x9dDa69d0CCdB06125a662070138800D4CE4F53b9")
	var data []byte
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	ts := types.Transactions{signedTx}
	rawTxBytes := ts.GetRlp(0)
	rawTxHex := hex.EncodeToString(rawTxBytes)

	return rawTxHex, nil
}

func TestMakeRawTxV2(ctx context.Context) (string, error) {
	const methoddata = `[
		{
		  "inputs": [],
		  "stateMutability": "nonpayable",
		  "type": "constructor"
		},
		{
		  "inputs": [],
		  "name": "COUNTER",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "inputs": [],
		  "name": "decrement",
		  "outputs": [],
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "inputs": [],
		  "name": "increment",
		  "outputs": [],
		  "stateMutability": "nonpayable",
		  "type": "function"
		}
	  ]`

	abi, err := abi.JSON(strings.NewReader(methoddata))
	if err != nil {
		return "", err
	}

	packed, err := abi.Pack("increment")
	if err != nil {
		return "", err
	}

	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return "", err
	}

	privateKey, err := crypto.HexToECDSA(os.Getenv("REPORTER_PK"))
	if err != nil {
		return "", err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", err
	}

	contractAddress := common.HexToAddress("0x93120927379723583c7a0dd2236fcb255e96949f")
	tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), uint64(90000), gasPrice, packed)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", err
	}

	ts := types.Transactions{signedTx}
	rawTxBytes := ts.GetRlp(0)
	rawTxHex := hex.EncodeToString(rawTxBytes)

	return rawTxHex, nil
}

func TestMakeFeeDelegatedRawTx(ctx context.Context) (*types.Transaction, error) {
	const methoddata = `[
		{
		  "inputs": [],
		  "stateMutability": "nonpayable",
		  "type": "constructor"
		},
		{
		  "inputs": [],
		  "name": "COUNTER",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "inputs": [],
		  "name": "decrement",
		  "outputs": [],
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "inputs": [],
		  "name": "increment",
		  "outputs": [],
		  "stateMutability": "nonpayable",
		  "type": "function"
		}
	  ]`

	abi, err := abi.JSON(strings.NewReader(methoddata))
	if err != nil {
		return nil, err
	}

	packed, err := abi.Pack("increment")
	if err != nil {
		return nil, err
	}

	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(os.Getenv("REPORTER_PK"))
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	feePayerPrivateKey, err := crypto.HexToECDSA(os.Getenv("TEST_FEE_PAYER_PK"))
	if err != nil {
		return nil, err
	}

	feePayerPublicKey := feePayerPrivateKey.Public()
	feePayerPublicKeyECDSA, ok := feePayerPublicKey.(*ecdsa.PublicKey)
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

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	contractAddress := common.HexToAddress("0x93120927379723583c7a0dd2236fcb255e96949f")

	txMap := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    nonce,
		types.TxValueKeyGasPrice: gasPrice,
		types.TxValueKeyGasLimit: uint64(90000),
		types.TxValueKeyTo:       contractAddress,
		types.TxValueKeyAmount:   big.NewInt(0),
		types.TxValueKeyFrom:     fromAddress,
		types.TxValueKeyData:     packed,
		types.TxValueKeyFeePayer: crypto.PubkeyToAddress(*feePayerPublicKeyECDSA),
	}
	unsigned, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, txMap)
	if err != nil {
		return nil, err
	}
	signedTx, err := types.SignTx(unsigned, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}

func GetRawTxHash(tx *types.Transaction) (string, error) {
	ts := types.Transactions{tx}
	rawTxBytes := ts.GetRlp(0)
	rawTxHex := hex.EncodeToString(rawTxBytes)

	return rawTxHex, nil
}

func SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return nil, err
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	feePayerPrivateKey, err := crypto.HexToECDSA(os.Getenv("TEST_FEE_PAYER_PK"))
	if err != nil {
		return nil, err
	}

	feePayerPublicKey := feePayerPrivateKey.Public()
	_, ok := feePayerPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	signedWithTxFeePayer, err := types.SignTxAsFeePayer(tx, types.NewEIP155Signer(chainID), feePayerPrivateKey)
	if err != nil {
		return nil, err
	}
	return signedWithTxFeePayer, nil
}

func TestSendRawTx(ctx context.Context, rawTx string) error {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return err
	}

	rawTxBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		return err
	}

	tx := new(types.Transaction)
	rlp.DecodeBytes(rawTxBytes, &tx)

	err = client.SendTransaction(ctx, tx)
	if err != nil {
		return err
	}

	fmt.Printf("tx sent: %s", tx.Hash().Hex())
	return nil
}

func GetPublicKey(pk string) (string, error) {
	pk = strings.TrimPrefix(pk, "0x")
	if len(pk) == 110 {
		return "", errors.New("klaytn wallet key is given instead of private key")
	}

	privateKeyECDSA, err := crypto.HexToECDSA(pk)
	if err != nil {
		return "", err
	}

	publicKey := privateKeyECDSA.Public()

	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", err
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address.String(), nil
}
