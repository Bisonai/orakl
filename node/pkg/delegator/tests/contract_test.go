// nolint:all
package tests

import (
	"testing"

	"bisonai.com/orakl/node/pkg/delegator/contract"
	"bisonai.com/orakl/node/pkg/delegator/utils"

	"github.com/stretchr/testify/assert"
)

func TestContractRead(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[[]contract.ContractDetailModel](appConfig.App, "/api/v1/contract", nil)
	assert.Nil(t, err)
	assert.Greater(t, len(readResult), 0)
}

func TestContractInsert(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockContract1 := contract.ContractInsertModel{
		Address: "0xbcd",
	}

	readResultBefore, err := utils.GetRequest[[]contract.ContractDetailModel](appConfig.App, "/api/v1/contract", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract", mockContract1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Address, mockContract1.Address)

	readResultAfter, err := utils.GetRequest[[]contract.ContractDetailModel](appConfig.App, "/api/v1/contract", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more contracts after insertion")

	//cleanup
	utils.QueryRowWithoutFiberCtx[contract.ContractDetailModel](appConfig.Postgres, contract.DeleteContract, map[string]any{"id": insertResult.ContractId})
}

func TestContractReadSingle(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract/"+insertedMockContract.ContractId.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.Address, insertedMockContract.Address)
}

func TestContractUpdate(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()
	readResultBefore, err := utils.GetRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract/"+insertedMockContract.ContractId.String(), nil)
	assert.Nil(t, err)

	_, err = utils.PatchRequest[contract.ContractModel](appConfig.App, "/api/v1/contract/"+insertedMockContract.ContractId.String(), map[string]any{"address": "0xc"})
	assert.Nil(t, err)
	readResultAfter, err := utils.GetRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract/"+insertedMockContract.ContractId.String(), nil)
	assert.Nil(t, err)
	assert.NotEqual(t, readResultBefore.Address, readResultAfter.Address)
	assert.Equal(t, readResultAfter.Address, "0xc")
}

func TestContractDelete(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockContract1 := contract.ContractInsertModel{
		Address: "0xbcd",
	}

	insertResult, err := utils.PostRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract", mockContract1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Address, mockContract1.Address)

	readResultBefore, err := utils.GetRequest[[]contract.ContractDetailModel](appConfig.App, "/api/v1/contract", nil)
	assert.Nil(t, err)

	_, err = utils.DeleteRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract/"+insertResult.ContractId.String(), nil)
	assert.Nil(t, err)

	readResultAfter, err := utils.GetRequest[[]contract.ContractDetailModel](appConfig.App, "/api/v1/contract", nil)
	assert.Nil(t, err)

	assert.Less(t, len(readResultAfter), len(readResultBefore))
}

func TestConnectReporter(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	_, err = utils.PostRequest[interface{}](appConfig.App, "/api/v1/contract/connectReporter", map[string]any{
		"contractId": insertedMockContract.ContractId,
		"reporterId": insertedMockReporter.ReporterId,
	})
	assert.Nil(t, err)

	readResult, err := utils.GetRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract/"+insertedMockContract.ContractId.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.Reporter[0], insertedMockReporter.Address)

	_, err = utils.PostRequest[interface{}](appConfig.App, "/api/v1/contract/disconnectReporter", map[string]any{
		"contractId": insertedMockContract.ContractId,
		"reporterId": insertedMockReporter.ReporterId,
	})
	assert.Nil(t, err)
	readResult, err = utils.GetRequest[contract.ContractDetailModel](appConfig.App, "/api/v1/contract/"+insertedMockContract.ContractId.String(), nil)
	assert.Nil(t, err)
	assert.Empty(t, readResult.Reporter)

	//add again
	_, err = utils.PostRequest[interface{}](appConfig.App, "/api/v1/contract/connectReporter", map[string]any{
		"contractId": insertedMockContract.ContractId,
		"reporterId": insertedMockReporter.ReporterId,
	})
	assert.Nil(t, err)
}
