//nolint:all
package tests

import (
	"testing"

	"bisonai.com/orakl/node/pkg/delegator/function"
	"bisonai.com/orakl/node/pkg/delegator/utils"

	"github.com/stretchr/testify/assert"
)

func TestFunctionRead(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[[]function.FunctionDetailModel](appConfig.App, "/api/v1/function", nil)
	assert.Nil(t, err)
	assert.Greater(t, len(readResult), 0)
}

func TestFunctionReadSingle(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[function.FunctionDetailModel](appConfig.App, "/api/v1/function/"+insertedMockFunction.FunctionId.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.Name, insertedMockFunction.Name)
}

func TestFunctionInsert(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockFunction1 := function.FunctionInsertModel{
		Name:       "submit(uint256,int256)",
		ContractId: insertedMockContract.ContractId,
	}

	readResultBefore, err := utils.GetRequest[[]function.FunctionModel](appConfig.App, "/api/v1/function", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[function.FunctionDetailModel](appConfig.App, "/api/v1/function", mockFunction1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Name, mockFunction1.Name)

	readResultAfter, err := utils.GetRequest[[]function.FunctionModel](appConfig.App, "/api/v1/function", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more functions after insertion")

	//cleanup
	utils.QueryRowWithoutFiberCtx[function.FunctionModel](appConfig.Postgres, function.DeleteFunctionById, map[string]any{"id": insertResult.FunctionId})
}

func TestFunctionUpdate(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockFunction1 := function.FunctionInsertModel{
		Name:       "submit(uint256,int256)",
		ContractId: insertedMockContract.ContractId,
	}

	insertResult, err := utils.PostRequest[function.FunctionModel](appConfig.App, "/api/v1/function", mockFunction1)
	assert.Nil(t, err)

	mockFunction1.Name = "submit(uint256,int256,uint256)"
	updateResult, err := utils.PatchRequest[function.FunctionModel](appConfig.App, "/api/v1/function/"+insertResult.FunctionId.String(), mockFunction1)
	assert.Nil(t, err)
	assert.Equal(t, updateResult.Name, mockFunction1.Name)

	//cleanup
	utils.QueryRowWithoutFiberCtx[function.FunctionModel](appConfig.Postgres, function.DeleteFunctionById, map[string]any{"id": insertResult.FunctionId})
}

func TestFunctionDelete(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockFunction1 := function.FunctionInsertModel{
		Name:       "submit(uint256,int256)",
		ContractId: insertedMockContract.ContractId,
	}

	insertResult, err := utils.PostRequest[function.FunctionModel](appConfig.App, "/api/v1/function", mockFunction1)
	assert.Nil(t, err)

	readResultBefore, err := utils.GetRequest[[]function.FunctionModel](appConfig.App, "/api/v1/function", nil)
	assert.Nil(t, err)

	_, err = utils.DeleteRequest[function.FunctionModel](appConfig.App, "/api/v1/function/"+insertResult.FunctionId.String(), nil)
	assert.Nil(t, err)

	readResultAfter, err := utils.GetRequest[[]function.FunctionModel](appConfig.App, "/api/v1/function", nil)
	assert.Nil(t, err)

	assert.Lessf(t, len(readResultAfter), len(readResultBefore), "expected to have less functions after deletion")
}
