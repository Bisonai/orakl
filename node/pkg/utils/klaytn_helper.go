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

	"github.com/klaytn/klaytn/accounts/abi"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/rlp"

	"github.com/rs/zerolog/log"
)

const DEFAULT_GAS_LIMIT = uint64(400000)

type ReporterModel struct {
	ID int64  `db:"id"`
	PK string `db:"pk"`
}

// not intended for massive concurrent submissions, use with caution
type TxHelper struct {
	client    *client.Client
	reporters []string
	chainID   *big.Int

	lastUsed int
}

func NewTxHelper(ctx context.Context) (*TxHelper, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return nil, err
	}

	chainID, err := getChainID(ctx, client)
	if err != nil {
		return nil, err
	}

	reporters, err := getReporters(ctx)
	if err != nil {
		return nil, err
	}

	if os.Getenv("REPORTER_PK") != "" {
		reporter := strings.TrimPrefix(os.Getenv("REPORTER_PK"), "0x")
		reporters = append(reporters, reporter)
	}

	if len(reporters) == 0 {
		return nil, errors.New("no reporters found")
	}

	return &TxHelper{
		client:    client,
		reporters: reporters,
		chainID:   chainID,
	}, nil
}

func (t *TxHelper) Close() {
	t.client.Close()
}

func (t *TxHelper) NextReporter() string {
	if len(t.reporters) == 0 {
		return ""
	}
	reporter := t.reporters[t.lastUsed]
	t.lastUsed = (t.lastUsed + 1) % len(t.reporters)
	return reporter
}

func (t *TxHelper) MakeDirectTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return makeDirectTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *TxHelper) MakeFeeDelegatedTx(ctx context.Context, contractAddressHex string, functionString string, args ...interface{}) (*types.Transaction, error) {
	return makeFeeDelegatedTx(ctx, t.client, contractAddressHex, t.NextReporter(), functionString, t.chainID, args...)
}

func (t *TxHelper) SubmitRawTx(ctx context.Context, tx *types.Transaction) error {
	return submitRawTx(ctx, t.client, tx)
}

func (t *TxHelper) SubmitRawTxString(ctx context.Context, rawTx string) error {
	return submitRawTxString(ctx, t.client, rawTx)
}

func (t *TxHelper) SignTxByFeePayer(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	return signTxByFeePayer(ctx, t.client, tx, t.chainID)
}

func TxToHash(tx *types.Transaction) string {
	ts := types.Transactions{tx}
	rawTxBytes := ts.GetRlp(0)
	return hex.EncodeToString(rawTxBytes)
}

func HashToTx(hash string) (*types.Transaction, error) {
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

func GenerateABI(functionString string) (*abi.ABI, error) {
	parts := strings.Split(functionString, "(")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid function string")
	}

	functionName := parts[0]
	arguments := strings.TrimRight(parts[1], ")")

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
    ]`, arguments, functionName)

	parsedABI, err := abi.JSON(strings.NewReader(json))
	if err != nil {
		return nil, err
	}

	return &parsedABI, nil
}

func getChainID(ctx context.Context, client *client.Client) (*big.Int, error) {
	return client.NetworkID(ctx)
}

func getReporters(ctx context.Context) ([]string, error) {
	reporterModels, err := db.QueryRows[ReporterModel](ctx, "SELECT * FROM reporters;", nil)
	if err != nil {
		return nil, err
	}

	reporters := make([]string, len(reporterModels))
	for i, reporter := range reporterModels {
		reporters[i] = reporter.PK
	}

	return reporters, nil
}

func makeDirectTx(ctx context.Context, client *client.Client, contractAddressHex string, reporter string, functionString string, chainID *big.Int, args ...interface{}) (*types.Transaction, error) {
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

	// defaults amount value to 0
	tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), DEFAULT_GAS_LIMIT, gasPrice, packed)
	return types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
}

func makeFeeDelegatedTx(ctx context.Context, client *client.Client, contractAddressHex string, reporter string, functionString string, chainID *big.Int, args ...interface{}) (*types.Transaction, error) {
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

	txMap := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    nonce,
		types.TxValueKeyGasPrice: gasPrice,
		types.TxValueKeyGasLimit: uint64(90000),
		types.TxValueKeyTo:       contractAddress,
		types.TxValueKeyAmount:   big.NewInt(0),
		types.TxValueKeyFrom:     fromAddress,
		types.TxValueKeyData:     packed,
		types.TxValueKeyFeePayer: common.Address{}, // assumes that reporter doesn't know fee payer address
	}

	unsigned, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, txMap)
	if err != nil {
		return nil, err
	}

	return types.SignTx(unsigned, types.NewEIP155Signer(chainID), privateKey)
}

func signTxByFeePayer(ctx context.Context, client *client.Client, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
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

	updatedTx, err := updateFeePayer(tx, crypto.PubkeyToAddress(*publicKeyECDSA))
	if err != nil {
		return nil, err
	}

	return types.SignTxAsFeePayer(updatedTx, types.NewEIP155Signer(chainID), feePayerPrivateKey)
}

func submitRawTx(ctx context.Context, client *client.Client, tx *types.Transaction) error {

	err := client.SendTransaction(ctx, tx)
	if err != nil {
		return err
	}

	log.Debug().Str("txHash", tx.Hash().Hex()).Msg("submitted")
	return nil
}

func submitRawTxString(ctx context.Context, client *client.Client, rawTx string) error {
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

	return submitRawTx(ctx, client, tx)
}

func updateFeePayer(tx *types.Transaction, feePayer common.Address) (*types.Transaction, error) {
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
