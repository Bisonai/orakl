package sign

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/delegator/store"
	"bisonai.com/miko/node/pkg/delegator/utils"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/kaiachain/kaia/rlp"
	"github.com/rs/zerolog/log"
)

const (
	defaultTransactionLimit = 100
	maxTransactionLimit     = 1000
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

type ContractModel struct {
	Address    string             `json:"address" db:"address"`
	ContractId *utils.CustomInt64 `json:"id" db:"contract_id"`
}

// initialize reloads the fee payer key from the configured source (env, Vault
// or GSM). A key passed in the request is refused: keys must not travel in
// URLs (access logs, proxies) and the key source is the only place to set one.
func initialize(c *fiber.Ctx) error {
	if c.Query("feePayerPrivateKey") != "" {
		return fiber.NewError(fiber.StatusBadRequest, "feePayerPrivateKey is not accepted; the key is loaded from the configured source")
	}

	pgx, err := utils.GetPgx(c)
	if err != nil {
		return err
	}
	if err := utils.InitFeePayerPK(c.Context(), pgx); err != nil {
		return err
	}
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

	// validate what gets signed: the decoded tx, not the request's to/input
	// fields, which are only stored and need not match RawTx
	to, selector, err := txTarget(transaction)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err = validateContractAddress(c, to); err != nil {
		logRejected(transaction, to, selector, err)
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}
	if err = validateFunction(c, to, selector); err != nil {
		logRejected(transaction, to, selector, err)
		return fiber.NewError(fiber.StatusForbidden, err.Error())
	}

	signedTransaction, err := signTxByFeePayerV2(c, transaction)
	if err != nil {
		return err
	}

	signedRawTxHash := TxToHash(signedTransaction)

	if s := utils.GetStore(c); s != nil {
		from, _ := transaction.From()
		record := store.Transaction{
			Timestamp: time.Now(),
			Sender:    from.Hex(),
			Contract:  to,
			Selector:  selector,
			Nonce:     transaction.Nonce(),
			TxHash:    signedTransaction.Hash().Hex(),
			Token:     tokenName(c),
		}
		if err := s.RecordTransaction(c.Context(), record); err != nil {
			log.Error().Err(err).Msg("failed to record transaction")
		}
		return c.JSON(SignModel{SignedRawTx: &signedRawTxHash})
	}

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
	// unbounded select loads the whole table into memory and can oom the process
	limit := c.QueryInt("limit", defaultTransactionLimit)
	if limit < 1 || limit > maxTransactionLimit {
		limit = defaultTransactionLimit
	}
	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	transactions, err := utils.QueryRows[SignModel](c, GetTransactions, map[string]any{"limit": limit, "offset": offset})
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

// txTarget returns the lowercase contract address and 4-byte function selector
// of a contract call. Contract creation and calls without a selector are refused.
func txTarget(tx *types.Transaction) (string, string, error) {
	to := tx.To()
	if to == nil {
		return "", "", errors.New("contract creation is not allowed")
	}
	data := tx.Data()
	if len(data) < 4 {
		return "", "", errors.New("transaction has no function selector")
	}
	return strings.ToLower(to.Hex()), "0x" + hex.EncodeToString(data[:4]), nil
}

func logRejected(tx *types.Transaction, to, selector string, reason error) {
	from, _ := tx.From()
	log.Warn().Str("from", strings.ToLower(from.Hex())).Str("to", to).Str("selector", selector).Err(reason).Msg("sign request rejected")
}

// validateFunction requires the selector to be registered for the contract.
// Approvals are cached for the process lifetime, like validateContractAddress.
func validateFunction(c *fiber.Ctx, address, selector string) error {
	validFunctions := c.Locals("validFunctions").(*sync.Map)
	key := address + selector
	if _, ok := validFunctions.Load(key); ok {
		return nil
	}
	if s := utils.GetStore(c); s != nil {
		allowed, err := s.FunctionAllowed(c.Context(), address, selector)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("not approved function %s for contract %s", selector, address)
		}
		validFunctions.Store(key, struct{}{})
		return nil
	}
	functions, err := utils.QueryRows[FunctionModel](c, GetFunctionByToAndEncodedName, map[string]any{"to": address, "encodedName": selector})
	if err != nil {
		return err
	}
	if len(functions) == 0 {
		return fmt.Errorf("not approved function %s for contract %s", selector, address)
	}
	validFunctions.Store(key, struct{}{})
	return nil
}

func validateContractAddress(c *fiber.Ctx, address string) error {
	validContracts := c.Locals("validContracts").(*sync.Map)
	if _, ok := validContracts.Load(address); ok {
		log.Info().Str("address", address).Msg("contract approved through cache")
		return nil
	} else if s := utils.GetStore(c); s != nil {
		allowed, err := s.ContractAllowed(c.Context(), address)
		if err != nil {
			return err
		}
		if !allowed {
			return fmt.Errorf("not approved contract address")
		}
		validContracts.Store(address, struct{}{})
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
	if err != nil {
		return nil, err
	}

	newTx.SetSignature(tx.GetTxInternalData().RawSignatureValues())
	return newTx, nil
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
