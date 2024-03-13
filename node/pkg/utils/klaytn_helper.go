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
	"time"

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

const (
	DEFAULT_GAS_LIMIT    = uint64(1200000)
	SELECT_WALLETS_QUERY = "SELECT * FROM wallets;"
)

type Wallet struct {
	ID int64  `db:"id"`
	PK string `db:"pk"`
}

type TxHelper struct {
	client       *client.Client
	wallets      []string
	chainID      *big.Int
	delegatorUrl string

	lastUsed int
}

type SignInsertPayload struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Input    string `json:"input"`
	Gas      string `json:"gas"`
	Value    string `json:"value"`
	ChainId  string `json:"chainId"`
	GasPrice string `json:"gasPrice"`
	Nonce    string `json:"nonce"`
	V        string `json:"v"`
	R        string `json:"r"`
	S        string `json:"s"`
	RawTx    string `json:"rawTx"`
}

type SignModel struct {
	Id          int64     `json:"id" db:"transaction_id"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	From        string    `json:"from" db:"from"`
	To          string    `json:"to" db:"to"`
	Input       string    `json:"input" db:"input"`
	Gas         string    `json:"gas" db:"gas"`
	Value       string    `json:"value" db:"value"`
	ChainId     string    `json:"chainId" db:"chainId"`
	GasPrice    string    `json:"gasPrice" db:"gasPrice"`
	Nonce       string    `json:"nonce" db:"nonce"`
	V           string    `json:"v" db:"v"`
	R           string    `json:"r" db:"r"`
	S           string    `json:"s" db:"s"`
	RawTx       string    `json:"rawTx" db:"rawTx"`
	SignedRawTx *string   `json:"signedRawTx" db:"signedRawTx"`
	Succeed     *bool     `json:"succeed" db:"succeed"`
	FunctionId  int64     `json:"functionId" db:"functionId"`
	ContractId  int64     `json:"contractId" db:"contractId"`
	ReporterId  int64     `json:"reporterId" db:"reporterId"`
}

func NewTxHelper(ctx context.Context) (*TxHelper, error) {
	wallets, err := getWallets(ctx)
	if err != nil {
		return nil, err
	}

	if os.Getenv("REPORTER_PK") != "" {
		wallet := strings.TrimPrefix(os.Getenv("REPORTER_PK"), "0x")
		wallets = append(wallets, wallet)
	}

	if len(wallets) == 0 {
		return nil, errors.New("no wallets found")
	}

	delegatorUrl := os.Getenv("DELEGATOR_URL")

	return newTxHelper(ctx, os.Getenv("PROVIDER_URL"), wallets, delegatorUrl)
}

func newTxHelper(ctx context.Context, providerUrl string, reporterPKs []string, delegatorUrl string) (*TxHelper, error) {
	if providerUrl == "" {
		return nil, errors.New("provider url not set")
	}

	client, err := client.Dial(providerUrl)
	if err != nil {
		return nil, err
	}

	chainID, err := getChainID(ctx, client)
	if err != nil {
		return nil, err
	}

	if len(reporterPKs) == 0 {
		return nil, errors.New("no wallets found")
	}

	return &TxHelper{
		client:       client,
		wallets:      reporterPKs,
		chainID:      chainID,
		delegatorUrl: delegatorUrl,
	}, nil
}

func (t *TxHelper) Close() {
	t.client.Close()
}

func (t *TxHelper) GetSignedFromDelegator(tx *types.Transaction) (*types.Transaction, error) {
	if t.delegatorUrl == "" {
		return nil, errors.New("delegator url not set")
	}

	payload, err := makeSignPayload(tx)
	if err != nil {
		return nil, err
	}

	result, err := UrlRequest[SignModel](t.delegatorUrl+"/api/v1/sign/volatile", "POST", payload, nil, "")
	if err != nil {
		log.Error().Err(err).Msg("failed to request sign from delegator")
		return nil, err
	}

	if result.SignedRawTx == nil {
		return nil, errors.New("no signed raw tx")
	}
	return HashToTx(*result.SignedRawTx)
}

func (t *TxHelper) NextReporter() string {
	if len(t.wallets) == 0 {
		return ""
	}
	reporter := t.wallets[t.lastUsed]
	t.lastUsed = (t.lastUsed + 1) % len(t.wallets)
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

func getChainID(ctx context.Context, client *client.Client) (*big.Int, error) {
	return client.NetworkID(ctx)
}

func getWallets(ctx context.Context) ([]string, error) {
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

func makeFeeDelegatedTx(ctx context.Context, client *client.Client, contractAddressHex string, reporter string, functionString string, chainID *big.Int, args ...interface{}) (*types.Transaction, error) {
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

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return err
	}
	log.Debug().Any("hash", receipt.TxHash).Msg("mined")
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

func makeSignPayload(tx *types.Transaction) (SignInsertPayload, error) {
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
