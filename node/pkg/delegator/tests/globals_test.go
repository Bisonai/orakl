//nolint:all
package tests

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/delegator/contract"
	"bisonai.com/miko/node/pkg/delegator/function"
	"bisonai.com/miko/node/pkg/delegator/organization"
	"bisonai.com/miko/node/pkg/delegator/reporter"
	"bisonai.com/miko/node/pkg/delegator/sign"
	"bisonai.com/miko/node/pkg/delegator/utils"

	"github.com/joho/godotenv"
	"github.com/kaiachain/kaia/blockchain/types"
	"github.com/kaiachain/kaia/common"
	"github.com/kaiachain/kaia/crypto"
	"github.com/kaiachain/kaia/rlp"
)

var mockTx *types.Transaction
var mockTxPayload sign.SignInsertPayload
var loadedChainId *big.Int

var testReporterPublicKey string
var testContractAddr = "0x93120927379723583c7a0dd2236fcb255e96949f"

var mockOrganization = organization.OrganizationInsertModel{
	Name: "test",
}
var mockFunction = function.FunctionInsertModel{
	Name: "increment()",
}

var insertedMockContract contract.ContractModel
var insertedMockOrganization organization.OrganizationModel
var insertedMockReporter reporter.ReporterModel
var insertedMockFunction function.FunctionModel
var insertedMockTx sign.SignModel

var appConfig utils.AppConfig

func makeMockTransaction() (*types.Transaction, error) {
	reporterPkString := os.Getenv("TEST_DELEGATOR_REPORTER_PK")
	reporterPkString = strings.TrimPrefix(reporterPkString, "0x")

	reporterPk, err := crypto.HexToECDSA(reporterPkString)
	if err != nil {
		return nil, err
	}

	_nonce, err := utils.GetNonce(common.HexToAddress(testReporterPublicKey))
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

	var feePayerPk string

	if feePayerPk = os.Getenv("DELEGATOR_FEEPAYER_PK"); feePayerPk == "" {
		return nil, fmt.Errorf("fee payer not initialized")
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

	_sig := mockTx.RawSignatureValues()
	r := "0x" + _sig[0].R.Text(16)
	s := "0x" + _sig[0].S.Text(16)
	v := "0x" + _sig[0].V.Text(16)

	_rawTxBytes, err := rlp.EncodeToBytes(mockTx)
	if err != nil {
		return sign.SignInsertPayload{}, err
	}

	rawTx := "0x" + hex.EncodeToString(_rawTxBytes)

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
	tmp_func, err := utils.QueryRowWithoutFiberCtx[function.FunctionModel](appConfig.Postgres, function.InsertFunction, map[string]any{"name": mockFunction.Name, "contract_id": insertedMockContract.ContractId.String(), "encodedName": _encodedName})
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
	function.Routes(v1)
	reporter.Routes(v1)
	sign.Routes(v1)
	organization.Routes(v1)

	return nil
}

func cleanup() {
	utils.QueryRowsWithoutFiberCtx[function.FunctionModel](appConfig.Postgres, function.DeleteFunctionById, map[string]any{"id": insertedMockFunction.FunctionId.String()})
	utils.QueryRowsWithoutFiberCtx[contract.ContractConnectModel](appConfig.Postgres, contract.DeleteContract, map[string]any{"id": insertedMockContract.ContractId.String()})
	utils.QueryRowsWithoutFiberCtx[reporter.ReporterModel](appConfig.Postgres, reporter.DeleteReporterById, map[string]any{"id": insertedMockReporter.ReporterId.String()})
	utils.QueryRowsWithoutFiberCtx[organization.OrganizationModel](appConfig.Postgres, organization.DeleteOrganization, map[string]any{"id": insertedMockOrganization.OrganizationId.String()})
	utils.QueryRowsWithoutFiberCtx[sign.SignModel](appConfig.Postgres, sign.DeleteTransactionById, map[string]any{"id": insertedMockTx.Id.String()})
}
