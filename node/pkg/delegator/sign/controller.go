package sign

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/delegator/utils"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/params"
	"github.com/klaytn/klaytn/rlp"
	"github.com/rs/zerolog/log"
)

type FeePayer struct {
	PrivateKey string `json:"privateKey" db:"privateKey"`
}

type SignInsertPayload struct {
	Timestamp   *utils.CustomDateTime `json:"timestamp" db:"timestamp"`
	From        string                `json:"from" validate:"required"`
	To          string                `json:"to" validate:"required"`
	Input       string                `json:"input" validate:"required"`
	Gas         string                `json:"gas" validate:"required"`
	Value       string                `json:"value" validate:"required"`
	ChainId     string                `json:"chainId" validate:"required"`
	GasPrice    string                `json:"gasPrice" validate:"required"`
	Nonce       string                `json:"nonce" validate:"required"`
	V           string                `json:"v" validate:"required"`
	R           string                `json:"r" validate:"required"`
	S           string                `json:"s" validate:"required"`
	RawTx       string                `json:"rawTx" validate:"required"`
	SignedRawTx *string               `json:"signedRawTx" db:"signedRawTx"`
	Succeed     *bool                 `json:"succeed" db:"succeed"`
	FunctionId  *utils.CustomInt64    `json:"functionId" db:"functionId"`
	ContractId  *utils.CustomInt64    `json:"contractId" db:"contractId"`
	ReporterId  *utils.CustomInt64    `json:"reporterId" db:"reporterId"`
}

type SignModel struct {
	Id          *utils.CustomInt64    `json:"id" db:"transaction_id"`
	Timestamp   *utils.CustomDateTime `json:"timestamp" db:"timestamp"`
	From        string                `json:"from" db:"from"`
	To          string                `json:"to" db:"to"`
	Input       string                `json:"input" db:"input"`
	Gas         string                `json:"gas" db:"gas"`
	Value       string                `json:"value" db:"value"`
	ChainId     string                `json:"chainId" db:"chainId"`
	GasPrice    string                `json:"gasPrice" db:"gasPrice"`
	Nonce       string                `json:"nonce" db:"nonce"`
	V           string                `json:"v" db:"v"`
	R           string                `json:"r" db:"r"`
	S           string                `json:"s" db:"s"`
	RawTx       string                `json:"rawTx" db:"rawTx"`
	SignedRawTx *string               `json:"signedRawTx" db:"signedRawTx"`
	Succeed     *bool                 `json:"succeed" db:"succeed"`
	FunctionId  *utils.CustomInt64    `json:"functionId" db:"functionId"`
	ContractId  *utils.CustomInt64    `json:"contractId" db:"contractId"`
	ReporterId  *utils.CustomInt64    `json:"reporterId" db:"reporterId"`
}

type FunctionModel struct {
	FunctionId  *utils.CustomInt64 `json:"id" db:"id"`
	Name        string             `json:"name" db:"name"`
	EncodedName string             `json:"encodedName" db:"encodedName"`
	ContractId  *utils.CustomInt64 `json:"contractId" db:"contractId"`
}

type ReporterModel struct {
	ReporterId     *utils.CustomInt64 `db:"id" json:"id"`
	Address        string             `db:"address" json:"address"`
	OrganizationId *utils.CustomInt64 `db:"organization_id" json:"organizationId"`
}

type ContractModel struct {
	Address    string             `json:"address" db:"address"`
	ContractId *utils.CustomInt64 `json:"id" db:"contract_id"`
}

func initialize(c *fiber.Ctx) error {
	pk := c.Query("feePayerPrivateKey", "")
	if pk == "" {
		pgx, err := utils.GetPgx(c)
		if err != nil {
			return err
		}
		err = utils.InitFeePayerPK(c.Context(), pgx)
		if err != nil {
			return err
		}
		return c.SendString("Initialized")
	}

	pk = strings.TrimPrefix(pk, "0x")

	utils.UpdateFeePayer(pk)

	return c.SendString("Initialized")
}

func getFeePayerAddress(c *fiber.Ctx) error {
	pk, err := utils.GetFeePayer(c)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		return err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("error casting public key to ECDSA")
	}

	result := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	if !strings.HasPrefix(result, "0x") {
		result = "0x" + result
	}

	return c.JSON(result)
}

func insert(c *fiber.Ctx) error {
	payload := new(SignInsertPayload)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	payload.Timestamp = &utils.CustomDateTime{Time: time.Now()}
	payload.From = strings.ToLower(payload.From)
	payload.To = strings.ToLower(payload.To)

	tx, err := insertTransaction(c, payload)
	if err != nil {
		return err
	}

	err = validateTransaction(c, tx)
	if err != nil {
		return err
	}
	err = signTxByFeePayer(c, tx)
	if err != nil {
		return err
	}

	result, err := updateTransaction(c, tx)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func onlySign(c *fiber.Ctx) error {
	payload := new(SignInsertPayload)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}
	transaction, err := HashToTx(payload.RawTx)
	if err != nil {
		return err
	}

	signedTransaction, err := signTxByFeePayerV2(c, transaction)
	if err != nil {
		return err
	}

	signedRawTx := TxToHash(signedTransaction)

	return c.JSON(SignModel{SignedRawTx: &signedRawTx})
}

func insertV2(c *fiber.Ctx) error {
	payload := new(SignInsertPayload)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	transaction, err := HashToTx(payload.RawTx)
	if err != nil {
		return err
	}

	validate := validator.New()
	if err = validate.Struct(payload); err != nil {
		return err
	}

	err = validateContractAddress(c, strings.ToLower(payload.To))
	if err != nil {
		return err
	}

	signedTransaction, err := signTxByFeePayerV2(c, transaction)
	if err != nil {
		return err
	}

	signedRawTxHash := TxToHash(signedTransaction)

	defer func() {
		succeed := true
		payload.Timestamp = &utils.CustomDateTime{Time: time.Now()}
		payload.From = strings.ToLower(payload.From)
		payload.To = strings.ToLower(payload.To)
		payload.Succeed = &succeed
		payload.SignedRawTx = &signedRawTxHash

		_, err := insertTransaction(c, payload)
		if err != nil {
			log.Error().Err(err).Msg("failed to insert transaction")
		}
	}()

	return c.JSON(SignModel{SignedRawTx: &signedRawTxHash})
}

func get(c *fiber.Ctx) error {
	transactions, err := utils.QueryRows[SignModel](c, GetTransactions, nil)
	if err != nil {
		return err
	}

	return c.JSON(transactions)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	transaction, err := utils.QueryRow[SignModel](c, GetTransactionById, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(transaction)
}

func validateTransaction(c *fiber.Ctx, tx *SignModel) error {
	encodedName := tx.Input[:10]
	contract, err := utils.QueryRow[ContractModel](c, GetContractByAddress, map[string]any{"address": tx.To})
	if err != nil {
		return err
	}
	functions, err := utils.QueryRows[FunctionModel](c, GetFunctionByToAndEncodedName, map[string]any{"to": tx.To, "encodedName": encodedName})
	if err != nil {
		return err
	}
	reporters, err := utils.QueryRows[ReporterModel](c, GetReporterByFromAndTo, map[string]any{"from": tx.From, "to": tx.To})
	if err != nil {
		return err
	}

	if len(functions) == 1 && len(reporters) == 1 && contract.ContractId != nil {
		tx.ReporterId = reporters[0].ReporterId
		tx.ContractId = contract.ContractId
		tx.FunctionId = functions[0].FunctionId
		return nil
	} else if len(functions) == 0 || len(reporters) == 0 || contract.ContractId == nil {
		return fmt.Errorf("not approved transaction")
	} else {
		return fmt.Errorf("unexpected result length")
	}
}

func validateContractAddress(c *fiber.Ctx, address string) error {
	validContracts := c.Locals("validContracts").(*sync.Map)
	if _, ok := validContracts.Load(address); ok {
		log.Info().Str("address", address).Msg("contract approved through cache")
		return nil
	} else {
		contract, err := utils.QueryRow[ContractModel](c, GetContractByAddress, map[string]any{"address": address})
		if err == nil && contract.ContractId != nil {
			validContracts.Store(address, struct{}{})
			return nil
		} else {
			return fmt.Errorf("not approved contract address")
		}
	}
}

func signTxByFeePayer(c *fiber.Ctx, tx *SignModel) error {

	pk, err := utils.GetFeePayer(c)
	if err != nil {
		log.Error().Err(err).Msg("failed to get fee payer")
		return err
	}

	feePayerKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert fee payer private key to ECDSA")
		return err
	}

	feePayerPublicKey, err := utils.GetPublicKey(pk)
	if err != nil {
		log.Error().Err(err).Msg("failed to get fee payer public key")
		return err
	}

	transaction, err := CreateUnsignedTx(tx, feePayerPublicKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to create unsigned transaction")
		return err
	}

	chainId, ok := new(big.Int).SetString(tx.ChainId[2:], 16)
	if !ok {
		log.Error().Err(err).Msg("failed to convert chainId to big.Int")
		return fmt.Errorf("failed to convert chainId to big.Int")
	}

	signedWithTxFeepayer, err := types.SignTxAsFeePayer(transaction, types.LatestSigner(&params.ChainConfig{ChainID: chainId}), feePayerKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to sign transaction with fee payer")
		return err
	}

	succeed := true
	rawTxBytes, _ := rlp.EncodeToBytes(signedWithTxFeepayer)
	rawTxHash := "0x" + hex.EncodeToString(rawTxBytes)

	tx.Succeed = &succeed
	tx.SignedRawTx = &rawTxHash

	return nil
}

func signTxByFeePayerV2(c *fiber.Ctx, tx *types.Transaction) (*types.Transaction, error) {
	pk, err := utils.GetFeePayer(c)
	if err != nil {
		log.Error().Err(err).Msg("failed to get fee payer")
		return nil, err
	}

	feePayerKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert fee payer private key to ECDSA")
		return nil, err
	}
	feePayerPublicKey := feePayerKey.Public()
	publicKeyECDSA, ok := feePayerPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	updatedTx, err := updateFeePayer(tx, crypto.PubkeyToAddress(*publicKeyECDSA))
	if err != nil {
		return nil, err
	}

	return types.SignTxAsFeePayer(updatedTx, types.NewEIP155Signer(tx.ChainId()), feePayerKey)
}

func CreateUnsignedTx(tx *SignModel, feePayerPublicKey string) (*types.Transaction, error) {
	nonce, err := strconv.ParseUint(tx.Nonce[2:], 16, 64)
	if err != nil {
		return nil, err
	}
	to := common.HexToAddress(tx.To)
	value, success := new(big.Int).SetString(tx.Value, 0)
	if !success {
		return nil, fmt.Errorf("failed to convert value to big.Int")
	}
	gas, err := strconv.ParseUint(tx.Gas[2:], 16, 64)
	if err != nil {
		return nil, err
	}
	gasPrice, success := new(big.Int).SetString(tx.GasPrice, 0)
	if !success {
		return nil, fmt.Errorf("failed to convert gas price to big.Int")
	}
	data, err := hex.DecodeString(tx.Input[2:])
	if err != nil {
		return nil, err
	}
	from := common.HexToAddress(tx.From)
	feePayer := common.HexToAddress(feePayerPublicKey)

	_map := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    nonce,
		types.TxValueKeyTo:       to,
		types.TxValueKeyAmount:   value,
		types.TxValueKeyGasLimit: gas,
		types.TxValueKeyGasPrice: gasPrice,
		types.TxValueKeyFrom:     from,
		types.TxValueKeyData:     data,
		types.TxValueKeyFeePayer: feePayer,
	}

	transaction, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, _map)
	if err != nil {
		return nil, err
	}

	r, _ := new(big.Int).SetString(tx.R, 0)
	s, _ := new(big.Int).SetString(tx.S, 0)
	v, _ := new(big.Int).SetString(tx.V, 0)

	transaction.SetSignature(types.TxSignatures([]*types.TxSignature{
		{
			R: r,
			S: s,
			V: v,
		},
	}))

	return transaction, nil
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

func insertTransaction(c *fiber.Ctx, payload *SignInsertPayload) (*SignModel, error) {
	transaction, err := utils.QueryRow[SignModel](c, InsertTransaction, map[string]any{
		"timestamp":   payload.Timestamp.String(),
		"from":        payload.From,
		"to":          payload.To,
		"input":       payload.Input,
		"gas":         payload.Gas,
		"value":       payload.Value,
		"chainId":     payload.ChainId,
		"gasPrice":    payload.GasPrice,
		"nonce":       payload.Nonce,
		"v":           payload.V,
		"r":           payload.R,
		"s":           payload.S,
		"rawTx":       payload.RawTx,
		"signedRawTx": payload.SignedRawTx,
		"succeed":     payload.Succeed,
	})
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func updateTransaction(c *fiber.Ctx, tx *SignModel) (*SignModel, error) {
	result, err := utils.QueryRow[SignModel](c, UpdateTransaction, map[string]any{
		"timestamp":   tx.Timestamp.String(),
		"from":        tx.From,
		"to":          tx.To,
		"input":       tx.Input,
		"gas":         tx.Gas,
		"value":       tx.Value,
		"chainId":     tx.ChainId,
		"gasPrice":    tx.GasPrice,
		"nonce":       tx.Nonce,
		"v":           tx.V,
		"r":           tx.R,
		"s":           tx.S,
		"rawTx":       tx.RawTx,
		"signedRawTx": tx.SignedRawTx,
		"succeed":     tx.Succeed,
		"functionId":  tx.FunctionId,
		"contractId":  tx.ContractId,
		"reporterId":  tx.ReporterId,
		"id":          tx.Id,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}
