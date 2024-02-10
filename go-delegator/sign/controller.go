package sign

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/go-delegator/utils"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/params"
	"github.com/klaytn/klaytn/rlp"
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
		feePayers, err := utils.QueryRows[FeePayer](c, GetFeePayers, nil)
		if err != nil {
			panic(err)
		}

		if len(feePayers) == 0 {
			panic("No fee payer found")
		} else if len(feePayers) == 1 {
			utils.UpdateFeePayer(feePayers[0].PrivateKey)
		} else {
			panic("Too many fee payers")
		}
	} else {
		utils.UpdateFeePayer(pk)

	}

	return c.SendString("Initialized")
}

func insert(c *fiber.Ctx) error {
	payload := new(SignInsertPayload)
	if err := c.BodyParser(payload); err != nil {
		panic(err)
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		panic(err)
	}

	payload.Timestamp = &utils.CustomDateTime{Time: time.Now()}
	payload.From = strings.ToLower(payload.From)
	payload.To = strings.ToLower(payload.To)

	transaction, err := utils.QueryRow[SignModel](c, InsertTransaction, map[string]any{
		"timestamp": payload.Timestamp.String(),
		"from":      payload.From,
		"to":        payload.To,
		"input":     payload.Input,
		"gas":       payload.Gas,
		"value":     payload.Value,
		"chainId":   payload.ChainId,
		"gasPrice":  payload.GasPrice,
		"nonce":     payload.Nonce,
		"v":         payload.V,
		"r":         payload.R,
		"s":         payload.S,
		"rawTx":     payload.RawTx,
	})
	if err != nil {
		panic(err)
	}
	err = validateTransaction(c, &transaction)
	if err != nil {
		panic(err)
	}
	err = signTxByFeePayer(c, &transaction)
	if err != nil {
		panic(err)
	}
	result, err := utils.QueryRow[SignModel](c, UpdateTransaction, map[string]any{
		"timestamp":   transaction.Timestamp.String(),
		"from":        transaction.From,
		"to":          transaction.To,
		"input":       transaction.Input,
		"gas":         transaction.Gas,
		"value":       transaction.Value,
		"chainId":     transaction.ChainId,
		"gasPrice":    transaction.GasPrice,
		"nonce":       transaction.Nonce,
		"v":           transaction.V,
		"r":           transaction.R,
		"s":           transaction.S,
		"rawTx":       transaction.RawTx,
		"signedRawTx": transaction.SignedRawTx,
		"succeed":     transaction.Succeed,
		"functionId":  transaction.FunctionId,
		"contractId":  transaction.ContractId,
		"reporterId":  transaction.ReporterId,
		"id":          transaction.Id,
	})
	if err != nil {
		panic(err)
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	transactions, err := utils.QueryRows[SignModel](c, GetTransactions, nil)
	if err != nil {
		panic(err)
	}

	return c.JSON(transactions)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	transaction, err := utils.QueryRow[SignModel](c, GetTransactionById, map[string]any{"id": id})
	if err != nil {
		panic(err)
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

func signTxByFeePayer(c *fiber.Ctx, tx *SignModel) error {
	pk, err := utils.GetFeePayer(c)
	if err != nil {
		return err
	}

	feePayerKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		return err
	}

	feePayerPublicKey, err := utils.GetPublicKey(pk)
	if err != nil {
		return err
	}

	transaction, err := CreateUnsignedTx(tx, feePayerPublicKey)
	if err != nil {
		return err
	}

	_chainId, ok := new(big.Int).SetString(tx.ChainId[2:], 16)
	if !ok {
		return fmt.Errorf("failed to convert chainId to big.Int")
	}

	signedWithTxFeepayer, err := types.SignTxAsFeePayer(transaction, types.LatestSigner(&params.ChainConfig{ChainID: _chainId}), feePayerKey)
	if err != nil {
		return err
	}

	_succeed := true
	rawTxBytes, _ := rlp.EncodeToBytes(signedWithTxFeepayer)
	_rawTxHash := "0x" + hex.EncodeToString(rawTxBytes)

	tx.Succeed = &_succeed
	tx.SignedRawTx = &_rawTxHash

	return nil
}

func CreateUnsignedTx(tx *SignModel, feePayerPublicKey string) (*types.Transaction, error) {
	_nonce, err := strconv.ParseUint(tx.Nonce[2:], 16, 64)
	if err != nil {
		return nil, err
	}
	_to := common.HexToAddress(tx.To)
	_value, success := new(big.Int).SetString(tx.Value, 0)
	if !success {
		return nil, fmt.Errorf("failed to convert value to big.Int")
	}
	_gas, err := strconv.ParseUint(tx.Gas[2:], 16, 64)
	if err != nil {
		return nil, err
	}
	_gasPrice, success := new(big.Int).SetString(tx.GasPrice, 0)
	if !success {
		return nil, fmt.Errorf("failed to convert gas price to big.Int")
	}
	_data, err := hex.DecodeString(tx.Input[2:])
	if err != nil {
		return nil, err
	}
	_from := common.HexToAddress(tx.From)
	_feePayer := common.HexToAddress(feePayerPublicKey)

	_map := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    _nonce,
		types.TxValueKeyTo:       _to,
		types.TxValueKeyAmount:   _value,
		types.TxValueKeyGasLimit: _gas,
		types.TxValueKeyGasPrice: _gasPrice,
		types.TxValueKeyFrom:     _from,
		types.TxValueKeyData:     _data,
		types.TxValueKeyFeePayer: _feePayer,
	}

	transaction, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, _map)
	if err != nil {
		return nil, err
	}

	_r, _ := new(big.Int).SetString(tx.R, 0)
	_s, _ := new(big.Int).SetString(tx.S, 0)
	_v, _ := new(big.Int).SetString(tx.V, 0)

	transaction.SetSignature(types.TxSignatures([]*types.TxSignature{
		{
			R: _r,
			S: _s,
			V: _v,
		},
	}))

	return transaction, nil
}
