package utils

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/encryptor"

	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/accounts/abi"
	"github.com/klaytn/klaytn/accounts/abi/bind"
	"github.com/klaytn/klaytn/blockchain/types"
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

func GenerateCallABI(functionName string, inputs string, outputs string) (*abi.ABI, error) {
	return generateABI(functionName, inputs, outputs, "nonpayable", false)
}

func GenerateViewABI(functionName string, inputs string, outputs string) (*abi.ABI, error) {
	return generateABI(functionName, inputs, outputs, "view", false)
}

func generateABI(functionName string, inputs string, outputs string, stateMutability string, payable bool) (*abi.ABI, error) {
	if functionName == "" {
		return nil, errorSentinel.ErrChainEmptyFuncStringParam
	}

	inputArgs := MakeAbiFuncAttribute(inputs)
	outputArgs := MakeAbiFuncAttribute(outputs)

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
		log.Error().Err(err).Msg("failed to parse abi")
		return nil, err
	}

	return &parsedABI, nil
}

func GenerateEventABI(eventName string, inputs string) (*abi.ABI, error) {
	if eventName == "" {
		return nil, errorSentinel.ErrChainEmptyEventNameStringParam
	}

	inputArgs := MakeAbiFuncAttribute(inputs)
	json := fmt.Sprintf(`[
		{
			"anonymous": false,
			"inputs": [%s],
			"name": "%s",
			"type": "event"
		}
	]`, inputArgs, eventName)

	parsedABI, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse abi")
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
		pk, err := encryptor.DecryptText(reporter.PK)
		if err != nil {
			log.Warn().Err(err).Msg("failed to decrypt pk")
			continue
		}
		wallet := strings.TrimPrefix(pk, "0x")
		wallets[i] = wallet
	}

	return wallets, nil
}

func InsertWallet(ctx context.Context, pk string) error {
	existingWallets, err := GetWallets(ctx)
	if err != nil {
		return err
	}

	for _, wallet := range existingWallets {
		if wallet == pk {
			return nil
		}
	}

	if os.Getenv("DATABASE_URL") == "" {
		log.Warn().Msg("DATABASE_URL is not set, skipping wallet insert")
		return nil
	}
	encryptedPk, err := encryptor.EncryptText(pk)
	if err != nil {
		return err
	}

	return db.QueryWithoutResult(ctx, "INSERT INTO wallets (pk) VALUES (@pk)", map[string]any{"pk": encryptedPk})
}

func GetChainID(ctx context.Context, client ClientInterface) (*big.Int, error) {
	return client.NetworkID(ctx)
}

func MakeDirectTx(ctx context.Context, client ClientInterface, contractAddressHex string, reporter string, functionString string, chainID *big.Int, nonce uint64, args ...interface{}) (*types.Transaction, error) {
	if client == nil {
		return nil, errorSentinel.ErrChainEmptyClientParam
	}

	if contractAddressHex == "" {
		return nil, errorSentinel.ErrChainEmptyAddressParam
	}

	if reporter == "" {
		return nil, errorSentinel.ErrChainEmptyReporterParam
	}

	if functionString == "" {
		return nil, errorSentinel.ErrChainEmptyFuncStringParam
	}

	if chainID == nil {
		return nil, errorSentinel.ErrChainEmptyChainIdParam
	}

	functionName, inputs, outputs, err := ParseMethodSignature(functionString)
	if err != nil {
		return nil, err
	}

	abi, err := GenerateCallABI(functionName, inputs, outputs)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate abi")
		return nil, err
	}

	packed, err := abi.Pack(functionName, args...)
	if err != nil {
		log.Error().Err(err).Msg("failed to pack abi")
		return nil, err
	}

	privateKey, err := crypto.HexToECDSA(reporter)
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

func MakeFeeDelegatedTx(ctx context.Context, client ClientInterface, contractAddressHex string, reporter string, functionString string, chainID *big.Int, nonce uint64, args ...interface{}) (*types.Transaction, error) {
	if client == nil {
		return nil, errorSentinel.ErrChainEmptyClientParam
	}

	if contractAddressHex == "" {
		return nil, errorSentinel.ErrChainEmptyAddressParam
	}

	if reporter == "" {
		return nil, errorSentinel.ErrChainEmptyReporterParam
	}

	if functionString == "" {
		return nil, errorSentinel.ErrChainEmptyFuncStringParam
	}

	if chainID == nil {
		return nil, errorSentinel.ErrChainEmptyChainIdParam
	}

	functionName, inputs, outputs, err := ParseMethodSignature(functionString)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse method signature")
		return nil, err
	}

	abi, err := GenerateCallABI(functionName, inputs, outputs)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate abi")
		return nil, err
	}

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
		return nil, errorSentinel.ErrChainPubKeyToECDSAFail
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

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

func SignTxByFeePayer(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	feePayer := strings.TrimPrefix(os.Getenv("TEST_FEE_PAYER_PK"), "0x")
	feePayerPrivateKey, err := crypto.HexToECDSA(feePayer)
	if err != nil {
		return nil, err
	}

	feePayerPublicKey := feePayerPrivateKey.Public()
	publicKeyECDSA, ok := feePayerPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errorSentinel.ErrChainPubKeyToECDSAFail
	}

	updatedTx, err := UpdateFeePayer(tx, crypto.PubkeyToAddress(*publicKeyECDSA))
	if err != nil {
		return nil, err
	}

	return types.SignTxAsFeePayer(updatedTx, types.NewEIP155Signer(chainID), feePayerPrivateKey)
}

func SubmitRawTx(ctx context.Context, client ClientInterface, tx *types.Transaction) error {
	log.Debug().Str("Player", "ChainHelper").Str("tx", tx.Hash().String()).Msg("submitting tx")
	err := client.SendTransaction(ctx, tx)
	if err != nil {
		log.Error().Str("Player", "ChainHelper").Err(err).Msg("failed to send tx")
		return err
	}
	log.Debug().Str("Player", "ChainHelper").Str("tx", tx.Hash().String()).Msg("tx sent")

	ctxWithTimeout, cancel := context.WithTimeout(ctx, DEFAULT_MINE_WAIT_TIME)
	defer cancel()

	log.Debug().Str("Player", "ChainHelper").Str("tx", tx.Hash().String()).Msg("waiting for tx to be mined")
	receipt, err := bind.WaitMined(ctxWithTimeout, client, tx)
	if err != nil {
		log.Error().Str("Player", "ChainHelper").Err(err).Msg("failed to wait for tx to be mined")
		return err
	}
	log.Debug().Str("Player", "ChainHelper").Str("tx", tx.Hash().String()).Msg("tx mined")

	if receipt.Status != 1 {
		log.Error().Str("tx", receipt.TxHash.String()).Msg("tx failed")
		return errorSentinel.ErrChainTransactionFail
	}

	log.Debug().Str("Player", "ChainHelper").Any("hash", receipt.TxHash).Msg("tx success")
	return nil
}

func SubmitRawTxString(ctx context.Context, client ClientInterface, rawTx string) error {
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
		return nil, errorSentinel.ErrChainEmptyToAddress
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

func ReadContract(ctx context.Context, client ClientInterface, functionString string, contractAddress string, args ...interface{}) (interface{}, error) {
	if client == nil {
		return nil, errorSentinel.ErrChainEmptyClientParam
	}

	if contractAddress == "" {
		return nil, errorSentinel.ErrChainEmptyAddressParam
	}

	if functionString == "" {
		return nil, errorSentinel.ErrChainEmptyFuncStringParam
	}

	functionName, inputs, outputs, err := ParseMethodSignature(functionString)
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

	result, err := client.CallContract(ctx, klaytn.CallMsg{
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

// reference: https://github.com/umbracle/ethgo/blob/main/abi/abi.go
var (
	funcRegexpWithReturn    = regexp.MustCompile(`(\w*)\s*\((.*)\)(.*)\s*returns\s*\((.*)\)`)
	funcRegexpWithoutReturn = regexp.MustCompile(`(\w*)\s*\((.*)\)(.*)`)
)

func ParseMethodSignature(name string) (string, string, string, error) {
	if name == "" {
		return "", "", "", errorSentinel.ErrChainEmptyNameParam
	}

	name = strings.Replace(name, "\n", " ", -1)
	name = strings.Replace(name, "\t", " ", -1)

	name = strings.TrimPrefix(name, "function ")
	name = strings.TrimSpace(name)

	var funcName, inputArgs, outputArgs string

	if strings.Contains(name, "returns") {
		matches := funcRegexpWithReturn.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return "", "", "", errorSentinel.ErrChainFailedToFindMethodSignatureMatch
		}
		funcName = strings.TrimSpace(matches[0][1])
		inputArgs = strings.TrimSpace(matches[0][2])
		outputArgs = strings.TrimSpace(matches[0][4])
	} else {
		matches := funcRegexpWithoutReturn.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return "", "", "", errorSentinel.ErrChainFailedToFindMethodSignatureMatch
		}
		funcName = strings.TrimSpace(matches[0][1])
		inputArgs = strings.TrimSpace(matches[0][2])
	}

	return funcName, inputArgs, outputArgs, nil
}

func MakeAbiFuncAttribute(args string) string {
	splittedArgs := strings.Split(args, ",")
	if len(splittedArgs) == 0 || splittedArgs[0] == "" {
		return ""
	}

	var parts []string
	for _, arg := range splittedArgs {
		arg = strings.TrimSpace(arg)
		part := strings.Split(arg, " ")

		if len(part) < 2 {
			parts = append(parts, fmt.Sprintf(`{"type":"%s"}`, part[0]))
		} else if part[1] == "indexed" && len(part) > 2 {
			parts = append(parts, fmt.Sprintf(`{"type":"%s","name":"%s","indexed":true}`, part[0], part[len(part)-1]))
		} else {
			parts = append(parts, fmt.Sprintf(`{"type":"%s","name":"%s"}`, part[0], part[len(part)-1]))
		}
	}
	return strings.Join(parts, ",\n")
}

func LoadProviderUrls(ctx context.Context, chainId int) ([]string, error) {
	providerUrls, err := db.QueryRows[ProviderUrl](ctx, SELECT_PROVIDER_URLS_QUERY, map[string]interface{}{"chain_id": chainId})
	if err != nil {
		return nil, err
	}
	result := make([]string, len(providerUrls))
	for i, providerUrl := range providerUrls {
		result[i] = providerUrl.Url
	}

	return result, nil
}

// reference: https://github.com/ethereum/go-ethereum/issues/19766#issuecomment-963442824
func ShouldRetryWithSwitchedJsonRPC(err error) bool {
	jsonErr, ok := err.(JsonRpcError)
	if ok {
		return IsJsonRpcFailureError(jsonErr.ErrorCode())
	}
	return false
}

// errorCode reference: https://www.jsonrpc.org/specification
func IsJsonRpcFailureError(errorCode int) bool {
	if errorCode == -32603 {
		return true
	}
	if errorCode <= -32000 && errorCode >= -32099 {
		return true
	}
	return false
}

func MakeValueSignature(value int64, timestamp int64, name string, pk *ecdsa.PrivateKey) ([]byte, error) {
	hash := Value2HashForSign(value, timestamp, name)
	signature, err := crypto.Sign(hash, pk)
	if err != nil {
		return nil, err
	}

	// Convert V from 0/1 to 27/28
	if signature[64] < 27 {
		signature[64] += 27
	}

	return signature, nil
}

func Value2HashForSign(value int64, timestamp int64, name string) []byte {
	bigIntVal := big.NewInt(value)
	bigIntTimestamp := big.NewInt(timestamp)

	valueBuf := make([]byte, 32)
	timestampBuf := make([]byte, 32)

	copy(valueBuf[32-len(bigIntVal.Bytes()):], bigIntVal.Bytes())
	copy(timestampBuf[32-len(bigIntTimestamp.Bytes()):], bigIntTimestamp.Bytes())

	feedHash := crypto.Keccak256([]byte(name))

	concatBytes := bytes.Join([][]byte{valueBuf, timestampBuf, feedHash}, nil)
	return crypto.Keccak256(concatBytes)
}

func StringToPk(pk string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(strings.TrimPrefix(pk, "0x"))
}

func StringPkToAddressHex(pk string) (string, error) {
	privateKey, err := StringToPk(pk)
	if err != nil {
		return "", err
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errorSentinel.ErrChainPubKeyToECDSAFail
	}
	result := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	if !strings.HasPrefix(result, "0x") {
		result = "0x" + result
	}

	return result, nil
}

func RecoverSigner(hash []byte, signature []byte) (address common.Address, err error) {
	if len(signature) != 65 {
		return common.Address{}, errorSentinel.ErrChainInvalidSignatureLength
	}

	// use copy to avoid modifying the original signature
	signatureCopy := make([]byte, len(signature))
	copy(signatureCopy, signature)
	signatureCopy[64] -= 27

	pubKey, err := crypto.SigToPub(hash, signatureCopy)
	if err != nil {
		return common.Address{}, err
	}

	address = crypto.PubkeyToAddress(*pubKey)

	return address, nil
}

func LoadSignerPk(ctx context.Context) (string, error) {
	signer, err := db.QueryRow[Wallet](ctx, LOAD_SIGNER, nil)
	if err != nil {
		return "", err
	}

	if signer.PK == "" {
		return "", errorSentinel.ErrChainSignerPKNotFound
	}

	pk, err := encryptor.DecryptText(signer.PK)
	if err != nil {
		log.Warn().Err(err).Msg("failed to decrypt pk")
		return "", err
	}
	wallet := strings.TrimPrefix(pk, "0x")
	return wallet, nil
}

func StoreSignerPk(ctx context.Context, pk string) error {
	encryptedPk, err := encryptor.EncryptText(pk)
	if err != nil {
		return err
	}
	return db.QueryWithoutResult(ctx, STORE_SIGNER, map[string]any{"pk": encryptedPk})
}

func NewPk(ctx context.Context) (*ecdsa.PrivateKey, string, error) {
	pk, err := crypto.GenerateKey()
	if err != nil {
		return nil, "", err
	}

	privateKeyBytes := crypto.FromECDSA(pk)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	return pk, privateKeyHex, nil
}

func GetNonceFromPk(ctx context.Context, pkString string, client ClientInterface) (uint64, error) {
	privateKey, err := crypto.HexToECDSA(pkString)
	if err != nil {
		return 0, nil
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return 0, errorSentinel.ErrChainPubKeyToECDSAFail
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return client.PendingNonceAt(ctx, fromAddress)
}
