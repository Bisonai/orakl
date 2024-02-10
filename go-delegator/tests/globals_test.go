package tests

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"bisonai.com/orakl/go-delegator/contract"
	"bisonai.com/orakl/go-delegator/function_"
	"bisonai.com/orakl/go-delegator/organization"
	"bisonai.com/orakl/go-delegator/reporter"
	"bisonai.com/orakl/go-delegator/sign"
	"bisonai.com/orakl/go-delegator/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/rlp"
)

var mockTx *types.Transaction
var mockTxPayload sign.SignInsertPayload
var loadedChainId *big.Int

var testReporterPublicKey string
var testContractAddr = "0x93120927379723583c7a0dd2236fcb255e96949f"

var mockOrganization = organization.OrganizationInsertModel{
	Name: "test",
}
var mockFunction = function_.FunctionInsertModel{
	Name: "increment()",
}

var insertedMockContract contract.ContractModel
var insertedMockOrganization organization.OrganizationModel
var insertedMockReporter reporter.ReporterModel
var insertedMockFunction function_.FunctionModel
var insertedMockTx sign.SignModel

var appConfig utils.AppConfig

func makeMockTransaction() (*types.Transaction, error) {
	reporterPkString := utils.LoadEnvVars()["TEST_DELEGATOR_REPORTER_PK"].(string)
	if len(reporterPkString) != 64 {
		return nil, fmt.Errorf("private key must be a 64-character hexadecimal string")
	}

	reporterPk, err := crypto.HexToECDSA(reporterPkString)
	if err != nil {
		return nil, err
	}

	_nonce, err := utils.GetNonce()
	if err != nil {
		return nil, err
	}

	_gasPrice, err := utils.GetGasPrice()
	if err != nil {
		return nil, err
	}

	_gas := uint64(90000)
	_to := common.HexToAddress(testContractAddr)
	_value := big.NewInt(0)
	_from := common.HexToAddress(testReporterPublicKey)

	// _data, err := hex.DecodeString(getEncodedMockfunction()[2:])
	_data := common.Hex2Bytes(getEncodedMockfunction()[2:])
	if err != nil {
		return nil, err
	}

	pgxPool, pgxError := pgxpool.New(context.Background(), utils.LoadEnvVars()["DATABASE_URL"].(string))
	if pgxError != nil {
		return nil, pgxError
	}

	feePayerPk, err := utils.LoadFeePayer(pgxPool)
	if err != nil {
		return nil, err
	}

	feePayerPublicKey, err := utils.GetPublicKey(feePayerPk)

	if err != nil {
		return nil, err
	}

	_feePayer := common.HexToAddress(feePayerPublicKey)
	_map := map[types.TxValueKeyType]interface{}{
		types.TxValueKeyNonce:    _nonce,
		types.TxValueKeyGasPrice: _gasPrice,
		types.TxValueKeyGasLimit: _gas,
		types.TxValueKeyTo:       _to,
		types.TxValueKeyAmount:   _value,
		types.TxValueKeyFrom:     _from,
		types.TxValueKeyData:     _data,
		types.TxValueKeyFeePayer: _feePayer,
	}

	transaction, err := types.NewTransactionWithMap(types.TxTypeFeeDelegatedSmartContractExecution, _map)
	if err != nil {
		return nil, err
	}

	err = transaction.Sign(types.LatestSignerForChainID(loadedChainId), reporterPk)
	// transaction, err = types.SignTx(transaction, types.LatestSigner(&params.ChainConfig{ChainID: loadedChainId}), reporterPk)
	fmt.Println(transaction.String())
	if err != nil {
		return nil, err
	}

	return transaction, err
}

func MakeMockTxPayload(mockTx *types.Transaction) (sign.SignInsertPayload, error) {
	_from, _ := mockTx.From()
	from := _from.Hex()
	to := mockTx.To().Hex()
	input := "0x" + common.Bytes2Hex(mockTx.Data())
	gas := fmt.Sprintf("0x%x", mockTx.Gas())
	value := "0x" + mockTx.Value().String()
	chainId := loadedChainId
	gasPrice := "0x" + mockTx.GasPrice().String()
	nonce := fmt.Sprintf("0x%x", mockTx.Nonce())

	if mockTx.GetTxInternalData().ValidateSignature() == false {
		fmt.Println("sig validation failed")
	}

	_sig := mockTx.RawSignatureValues()
	r := "0x" + _sig[0].R.Text(16)
	s := "0x" + _sig[0].S.Text(16)
	v := "0x" + _sig[0].V.Text(16)

	_rawTxBytes, err := rlp.EncodeToBytes(mockTx)
	if err != nil {
		return sign.SignInsertPayload{}, err
	}

	rawTx := "0x" + hex.EncodeToString(_rawTxBytes)
	fmt.Println(rawTx)

	return sign.SignInsertPayload{
		From:     strings.ToLower(from),
		To:       strings.ToLower(to),
		Input:    input,
		Gas:      gas,
		Value:    value,
		ChainId:  "0x" + chainId.Text(16),
		GasPrice: gasPrice,
		Nonce:    nonce,
		R:        r,
		S:        s,
		V:        v,
		RawTx:    rawTx,
	}, nil
}

func getEncodedMockfunction() string {
	hash := crypto.Keccak256([]byte("increment()"))
	return "0x" + hex.EncodeToString(hash[:4])
}

func setup() error {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("env file is not found, continuing without .env file")
	}

	appConfig, err = utils.Setup()
	if err != nil {
		return err
	}

	mockTx := mockTxPayload

	if err != nil {
		return err
	}

	tmp_org, err := utils.QueryRowWithoutFiberCtx[organization.OrganizationModel](appConfig.Postgres, organization.InsertOrganization, map[string]any{"name": mockOrganization.Name})
	if err != nil {
		return err
	}
	insertedMockOrganization = tmp_org

	tmp_rep, err := utils.QueryRowWithoutFiberCtx[reporter.ReporterModel](appConfig.Postgres, reporter.InsertReporter, map[string]any{"address": strings.ToLower(testReporterPublicKey), "organizationId": insertedMockOrganization.OrganizationId.String()})
	if err != nil {
		return err
	}
	insertedMockReporter = tmp_rep

	tmp_con, err := utils.QueryRowWithoutFiberCtx[contract.ContractModel](appConfig.Postgres, contract.InsertContract, map[string]any{"address": strings.ToLower(testContractAddr)})
	if err != nil {
		return err
	}
	insertedMockContract = tmp_con

	hash := crypto.Keccak256([]byte(mockFunction.Name))
	_encodedName := "0x" + hex.EncodeToString(hash[:4])
	tmp_func, err := utils.QueryRowWithoutFiberCtx[function_.FunctionModel](appConfig.Postgres, function_.InsertFunction, map[string]any{"name": mockFunction.Name, "contract_id": insertedMockContract.ContractId.String(), "encodedName": _encodedName})
	if err != nil {
		return err
	}
	insertedMockFunction = tmp_func

	_, err = utils.QueryRowsWithoutFiberCtx[interface{}](appConfig.Postgres, contract.ConnectReporter, map[string]any{"contractId": insertedMockContract.ContractId.String(), "reporterId": insertedMockReporter.ReporterId.String()})
	if err != nil {
		return err
	}

	tmp_tx, err := utils.QueryRowWithoutFiberCtx[sign.SignModel](appConfig.Postgres, sign.InsertTransaction, map[string]any{
		"timestamp": utils.CustomDateTime{Time: time.Now()}.String(),
		"from":      mockTx.From,
		"to":        mockTx.To,
		"input":     mockTx.Input,
		"gas":       mockTx.Gas,
		"value":     mockTx.Value,
		"chainId":   mockTx.ChainId,
		"gasPrice":  mockTx.GasPrice,
		"nonce":     mockTx.Nonce,
		"v":         mockTx.V,
		"r":         mockTx.R,
		"s":         mockTx.S,
		"rawTx":     mockTx.RawTx,
	})
	if err != nil {
		return err
	}

	insertedMockTx = tmp_tx

	v1 := appConfig.App.Group("/api/v1")
	contract.Routes(v1)
	function_.Routes(v1)
	reporter.Routes(v1)
	sign.Routes(v1)
	organization.Routes(v1)

	return nil
}

func cleanup() {
	utils.QueryRowsWithoutFiberCtx[function_.FunctionModel](appConfig.Postgres, function_.DeleteFunctionById, map[string]any{"id": insertedMockFunction.FunctionId.String()})
	utils.QueryRowsWithoutFiberCtx[contract.ContractConnectModel](appConfig.Postgres, contract.DeleteContract, map[string]any{"id": insertedMockContract.ContractId.String()})
	utils.QueryRowsWithoutFiberCtx[reporter.ReporterModel](appConfig.Postgres, reporter.DeleteReporterById, map[string]any{"id": insertedMockReporter.ReporterId.String()})
	utils.QueryRowsWithoutFiberCtx[organization.OrganizationModel](appConfig.Postgres, organization.DeleteOrganization, map[string]any{"id": insertedMockOrganization.OrganizationId.String()})
	utils.QueryRowsWithoutFiberCtx[sign.SignModel](appConfig.Postgres, sign.DeleteTransactionById, map[string]any{"id": insertedMockTx.Id.String()})
}

/*
*types.Transaction {
	data: github.com/klaytn/klaytn/blockchain/types.TxInternalData(*github.com/klaytn/klaytn/blockchain/types.TxInternalDataFeeDelegatedSmartContractExecution) *{
		AccountNonce: 0,
		Price: *(*"math/big.Int")(0x140003ac000),
		GasLimit: 90000,
		Recipient: github.com/klaytn/klaytn/common.Address [147,18,9,39,55,151,35,88,60,122,13,210,35,111,203,37,94,150,148,159],
		Amount: *(*"math/big.Int")(0x140003ac020),
		From: github.com/klaytn/klaytn/common.Address [157,218,105,208,204,219,6,18,90,102,32,112,19,136,0,212,206,79,83,185],
		Payload: []uint8 len: 4, cap: 4, [208,157,224,138],
		TxSignatures: github.com/klaytn/klaytn/blockchain/types.TxSignatures len: 1, cap: 4, [*(*"github.com/klaytn/klaytn/blockchain/types.TxSignature")(0x1400098a030)],
		FeePayer: github.com/klaytn/klaytn/common.Address [0,38,222,52,82,38,39,197,218,43,106,86,24,20,122,145,83,193,36,58],
		FeePayerSignatures: github.com/klaytn/klaytn/blockchain/types.TxSignatures len: 1, cap: 4, [*(*"github.com/klaytn/klaytn/blockchain/types.TxSignature")(0x1400098a078)],
		Hash: *github.com/klaytn/klaytn/common.Hash [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
	},
	time: time.Time(2024-02-12T19:39:58+09:00, +135892860293){
		wall: 13937122694850669760,
		ext: 135892860293,
		loc: *(*time.Location)(0x103552280)
	},
	hash: sync/atomic.Value {v: interface {} nil},
	size: sync/atomic.Value {v: interface {}(github.com/klaytn/klaytn/common.StorageSize) *(*interface {})(0x14000984038)},
	from: sync/atomic.Value {v: interface {} nil},
	feePayer: sync/atomic.Value {v: interface {} nil},
	senderTxHash: sync/atomic.Value {v: interface {} nil},
	validatedSender: github.com/klaytn/klaytn/common.Address [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
	validatedFeePayer: github.com/klaytn/klaytn/common.Address [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
	validatedIntrinsicGas: 0,
	checkNonce: false,
	markedUnexecutable: 0,
	mu: sync.RWMutex {
		w: (*sync.Mutex)(0x140009840b0),
		writerSem: 0,
		readerSem: 0,
		readerCount: (*"sync/atomic.Int32")(0x140009840c0),
		readerWait: (*"sync/atomic.Int32")(0x140009840c4)
	}
}
-- after fee payer signed
types.Transaction {
	data: github.com/klaytn/klaytn/blockchain/types.TxInternalData(*github.com/klaytn/klaytn/blockchain/types.TxInternalDataFeeDelegatedSmartContractExecution) *{
		AccountNonce: 0,
		Price: *(*"math/big.Int")(0x140000bb220),
		GasLimit: 90000,
		Recipient: github.com/klaytn/klaytn/common.Address [147,18,9,39,55,151,35,88,60,122,13,210,35,111,203,37,94,150,148,159],
		Amount: *(*"math/big.Int")(0x140000bb240),
		From: github.com/klaytn/klaytn/common.Address [157,218,105,208,204,219,6,18,90,102,32,112,19,136,0,212,206,79,83,185],
		Payload: []uint8 len: 4, cap: 8, [208,157,224,138],
		TxSignatures: github.com/klaytn/klaytn/blockchain/types.TxSignatures len: 1, cap: 1, [*(*"github.com/klaytn/klaytn/blockchain/types.TxSignature")(0x1400000f1b8)],
		FeePayer: github.com/klaytn/klaytn/common.Address [0,38,222,52,82,38,39,197,218,43,106,86,24,20,122,145,83,193,36,58],
		FeePayerSignatures: github.com/klaytn/klaytn/blockchain/types.TxSignatures len: 1, cap: 1, [*(*"github.com/klaytn/klaytn/blockchain/types.TxSignature")(0x1400000f218)],
		Hash: *github.com/klaytn/klaytn/common.Hash [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
	},
	time: time.Time(2024-02-12T19:44:09+09:00, +16021792418){
		wall: 13937122964243807584,
		ext: 16021792418,
		loc: *(*time.Location)(0x10479e280)
	},
	hash: sync/atomic.Value {v: interface {} nil},
	size: sync/atomic.Value {v: interface {} nil},
	from: sync/atomic.Value {v: interface {} nil},
	feePayer: sync/atomic.Value {v: interface {} nil},
	senderTxHash: sync/atomic.Value {v: interface {} nil},
	validatedSender: github.com/klaytn/klaytn/common.Address [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
	validatedFeePayer: github.com/klaytn/klaytn/common.Address [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
	validatedIntrinsicGas: 0,
	checkNonce: false,
	markedUnexecutable: 0,
	mu: sync.RWMutex {
		w: (*sync.Mutex)(0x1400046d430),
		writerSem: 0,
		readerSem: 0,
		readerCount: (*"sync/atomic.Int32")(0x1400046d440),
		readerWait: (*"sync/atomic.Int32")(0x1400046d444)
	}
}

-- after raw tx decoding
types.Transaction {
	data: github.com/klaytn/klaytn/blockchain/types.TxInternalData(*github.com/klaytn/klaytn/blockchain/types.TxInternalDataFeeDelegatedSmartContractExecution) *{
		AccountNonce: 0,
		Price: *(*"math/big.Int")(0x140003ac520),
		GasLimit: 90000,
		Recipient: github.com/klaytn/klaytn/common.Address [147,18,9,39,55,151,35,88,60,122,13,210,35,111,203,37,94,150,148,159],
		Amount: *(*"math/big.Int")(0x140003ac540),
		From: github.com/klaytn/klaytn/common.Address [157,218,105,208,204,219,6,18,90,102,32,112,19,136,0,212,206,79,83,185],
		Payload: []uint8 len: 4, cap: 4, [208,157,224,138],
		TxSignatures: github.com/klaytn/klaytn/blockchain/types.TxSignatures len: 1, cap: 4, [*(*"github.com/klaytn/klaytn/blockchain/types.TxSignature")(0x1400000e378)],
		FeePayer: github.com/klaytn/klaytn/common.Address [0,38,222,52,82,38,39,197,218,43,106,86,24,20,122,145,83,193,36,58],
		FeePayerSignatures: github.com/klaytn/klaytn/blockchain/types.TxSignatures len: 1, cap: 4, [*(*"github.com/klaytn/klaytn/blockchain/types.TxSignature")(0x1400000e3d8)],
		Hash: *github.com/klaytn/klaytn/common.Hash [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]
	},
	time: time.Time(2024-02-12T19:48:20+09:00, +266511206668) {
		wall: 13937123233243769408,
		ext: 266511206668,
		loc: *(*time.Location)(0x10479e280)
	},
	hash: sync/atomic.Value {v: interface {} nil},
	size: sync/atomic.Value {v: interface {}(github.com/klaytn/klaytn/common.StorageSize) *(*interface {})(0x140007ca378)},
	from: sync/atomic.Value {v: interface {} nil},
	feePayer: sync/atomic.Value {v: interface {} nil},
	senderTxHash: sync/atomic.Value {v: interface {} nil},
	validatedSender: github.com/klaytn/klaytn/common.Address [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
	validatedFeePayer: github.com/klaytn/klaytn/common.Address [0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],
	validatedIntrinsicGas: 0,
	checkNonce: false,
	markedUnexecutable: 0,
	mu: sync.RWMutex {
		w: (*sync.Mutex)(0x140007ca3f0),
		writerSem: 0,
		readerSem: 0,
		readerCount: (*"sync/atomic.Int32")(0x140007ca400),
		readerWait: (*"sync/atomic.Int32")(0x140007ca404)
	}
}
*/
