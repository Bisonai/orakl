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

	"bisonai.com/orakl/node/pkg/db"

	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/accounts/abi"
	"github.com/klaytn/klaytn/accounts/abi/bind"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/rlp"

	"github.com/rs/zerolog/log"
)

func MakePayload(tx *types.Transaction) (SignInsertPayload, error) {
	rawFrom, err := tx.From()
	if err != nil {
		log.Error().Err(err).Msg("failed to get from")
		return SignInsertPayload{}, err
	}

	from := strings.ToLower(rawFrom.Hex())
	to := strings.ToLower(tx.To().Hex())
	input := "0x" + hex.EncodeToString(tx.Data())
	gas := fmt.Sprintf("0x%x", tx.Gas())
	value := "0x" + tx.Value().String()
	chainId := tx.ChainId()
	gasPrice := "0x" + tx.GasPrice().String()
	nonce := fmt.Sprintf("0x%x", tx.Nonce())

	sig := tx.RawSignatureValues()
	r := "0x" + sig[0].R.Text(16)
	s := "0x" + sig[0].S.Text(16)
	v := "0x" + sig[0].V.Text(16)

	rawTx := "0x" + TxToHash(tx)

	payload := SignInsertPayload{
		From:     from,
		To:       to,
		Input:    input,
		Gas:      gas,
		Value:    value,
		ChainId:  "0x" + chainId.Text(16),
		GasPrice: gasPrice,
		Nonce:    nonce,
		V:        r,
		R:        s,
		S:        v,
		RawTx:    rawTx,
	}

	return payload, nil
}

func TxToHash(tx *types.Transaction) string {
	ts := types.Transactions{tx}
	rawTxBytes := ts.GetRlp(0)
	return hex.EncodeToString(rawTxBytes)
}

func HashToTx(hash string) (*types.Transaction, error) {
	hash = strings.TrimPrefix(hash, "0x")
	rawTxBytes, err := hex.DecodeString(hash)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode hash")
		return nil, err
	}

	tx := new(types.Transaction)
	err = rlp.DecodeBytes(rawTxBytes, &tx)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode raw tx")
		return nil, err
	}
	return tx, nil
}

func ConvertFunctionParameters(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	parts := strings.Split(input, ",")
	var outputParts []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		paramType := strings.Split(part, " ")[0]
		outputParts = append(outputParts, fmt.Sprintf(`{
            "type": "%s"
        }`, paramType))
	}
	return strings.Join(outputParts, ",\n")
}

func GenerateABI(functionString string) (*abi.ABI, error) {
	parts := strings.Split(functionString, "(")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid function string")
	}

	functionName := parts[0]
	arguments := strings.TrimRight(parts[1], ")")
	convertedArgs := ConvertFunctionParameters(arguments)
	json := fmt.Sprintf(`[
        {
            "constant": false,
            "inputs": [%s],
            "name": "%s",
            "outputs": [],
            "payable": false,
            "stateMutability": "nonpayable",
            "type": "function"
        }
    ]`, convertedArgs, functionName)

	parsedABI, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		return nil, err
	}

	return &parsedABI, nil
}

func GetWallets(ctx context.Context) ([]string, error) {
	reporterModels, err := db.QueryRows[Wallet](ctx, SELECT_WALLETS_QUERY, nil)
	if err != nil {
		return nil, err
	}

	wallets := make([]string, len(reporterModels))
	for i, reporter := range reporterModels {
		wallets[i] = reporter.PK
	}

	return wallets, nil
}

func GetChainID(ctx context.Context, client *client.Client) (*big.Int, error) {
	return client.NetworkID(ctx)
}

func MakeDirectTx(ctx context.Context, client *client.Client, contractAddressHex string, reporter string, functionString string, chainID *big.Int, args ...interface{}) (*types.Transaction, error) {
	abi, err := GenerateABI(functionString)
	if err != nil {
		return nil, err
	}

	functionName := strings.Split(functionString, "(")[0]
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

	estimatedGas, err := client.EstimateGas(ctx, klaytn.CallMsg{
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

func MakeFeeDelegatedTx(ctx context.Context, client *client.Client, contractAddressHex string, reporter string, functionString string, chainID *big.Int, args ...interface{}) (*types.Transaction, error) {
	abi, err := GenerateABI(functionString)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate abi")
		return nil, err
	}

	functionName := strings.Split(functionString, "(")[0]
	packed, err := abi.Pack(functionName, args...)
	if err != nil {
		log.Error().Err(err).Msg("failed to pack abi")
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

	estimatedGas, err := client.EstimateGas(ctx, klaytn.CallMsg{
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

	txMap := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    nonce,
		types.TxValueKeyGasPrice: gasPrice,
		types.TxValueKeyGasLimit: estimatedGas,
		types.TxValueKeyTo:       contractAddress,
		types.TxValueKeyAmount:   big.NewInt(0),
		types.TxValueKeyFrom:     fromAddress,
		types.TxValueKeyData:     packed,
		types.TxValueKeyFeePayer: common.Address{},
	}

	unsigned, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, txMap)
	if err != nil {
		return nil, err
	}

	return types.SignTx(unsigned, types.NewEIP155Signer(chainID), privateKey)
}

func SignTxByFeePayer(ctx context.Context, client *client.Client, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	feePayer := strings.TrimPrefix(os.Getenv("TEST_FEE_PAYER_PK"), "0x")
	feePayerPrivateKey, err := crypto.HexToECDSA(feePayer)
	if err != nil {
		return nil, err
	}

	feePayerPublicKey := feePayerPrivateKey.Public()
	publicKeyECDSA, ok := feePayerPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	updatedTx, err := UpdateFeePayer(tx, crypto.PubkeyToAddress(*publicKeyECDSA))
	if err != nil {
		return nil, err
	}

	return types.SignTxAsFeePayer(updatedTx, types.NewEIP155Signer(chainID), feePayerPrivateKey)
}

func SubmitRawTx(ctx context.Context, client *client.Client, tx *types.Transaction) error {
	err := client.SendTransaction(ctx, tx)
	if err != nil {
		return err
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return err
	}

	if receipt.Status != 1 {
		log.Error().Str("tx", receipt.TxHash.String()).Msg("tx failed")
		return fmt.Errorf("transaction failed (hash: %s), status: %d", receipt.TxHash.String(), receipt.Status)
	}

	log.Debug().Any("hash", receipt.TxHash).Msg("tx success")
	return nil
}

func SubmitRawTxString(ctx context.Context, client *client.Client, rawTx string) error {
	rawTxBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		return err
	}

	tx := new(types.Transaction)
	err = rlp.DecodeBytes(rawTxBytes, &tx)
	if err != nil {
		log.Error().Err(err).Msg("failed to decode raw tx")
		return err
	}

	return SubmitRawTx(ctx, client, tx)
}

func UpdateFeePayer(tx *types.Transaction, feePayer common.Address) (*types.Transaction, error) {
	from, err := tx.From()
	if err != nil {
		return nil, err
	}

	to := tx.To()
	if to == nil {
		return nil, errors.New("to address is nil")
	}

	remap := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    tx.Nonce(),
		types.TxValueKeyGasPrice: tx.GasPrice(),
		types.TxValueKeyGasLimit: tx.Gas(),
		types.TxValueKeyTo:       *to,
		types.TxValueKeyAmount:   tx.Value(),
		types.TxValueKeyFrom:     from,
		types.TxValueKeyData:     tx.Data(),
		types.TxValueKeyFeePayer: feePayer,
	}

	newTx, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, remap)

	newTx.SetSignature(tx.GetTxInternalData().RawSignatureValues())
	return newTx, err
}
