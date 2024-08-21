//nolint:all
package tests

import (
	"testing"

	"bisonai.com/miko/node/pkg/delegator/reporter"
	"bisonai.com/miko/node/pkg/delegator/utils"

	"github.com/stretchr/testify/assert"
)

func TestReporterRead(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)
	assert.Greater(t, len(readResult), 0)
}

func TestReporterReadSingle(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter/"+insertedMockReporter.ReporterId.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.Address, insertedMockReporter.Address)
}

func TestReporterInsert(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockReporter1 := reporter.ReporterInsertModel{
		Address:        "0x123",
		OrganizationId: &insertedMockOrganization.OrganizationId,
	}

	readResultBefore, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[reporter.ReporterModel](appConfig.App, "/api/v1/reporter", mockReporter1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Address, mockReporter1.Address)

	readResultAfter, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more reporters after insertion")

	//cleanup
	utils.QueryRowWithoutFiberCtx[reporter.ReporterModel](appConfig.Postgres, reporter.DeleteReporterById, map[string]any{"id": insertResult.ReporterId})
}

func TestReporterUpdate(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockReporter1 := reporter.ReporterInsertModel{
		Address:        "0x123",
		OrganizationId: &insertedMockOrganization.OrganizationId,
	}

	readResultBefore, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[reporter.ReporterModel](appConfig.App, "/api/v1/reporter", mockReporter1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Address, mockReporter1.Address)

	readResultAfter, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more reporters after insertion")

	mockReporter1.Address = "0x456"
	updateResult, err := utils.PatchRequest[reporter.ReporterModel](appConfig.App, "/api/v1/reporter/"+insertResult.ReporterId.String(), mockReporter1)
	assert.Nil(t, err)
	assert.Equal(t, updateResult.Address, mockReporter1.Address)

	//cleanup
	utils.QueryRowWithoutFiberCtx[reporter.ReporterModel](appConfig.Postgres, reporter.DeleteReporterById, map[string]any{"id": insertResult.ReporterId})
}

func TestReporterDelete(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockReporter1 := reporter.ReporterInsertModel{
		Address:        "0x123",
		OrganizationId: &insertedMockOrganization.OrganizationId,
	}

	insertResult, err := utils.PostRequest[reporter.ReporterModel](appConfig.App, "/api/v1/reporter", mockReporter1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Address, mockReporter1.Address)

	readResultBefore, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)

	deleteResult, err := utils.DeleteRequest[reporter.ReporterModel](appConfig.App, "/api/v1/reporter/"+insertResult.ReporterId.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, deleteResult.Address, mockReporter1.Address)

	readResultAfter, err := utils.GetRequest[[]reporter.ReporterDetailModel](appConfig.App, "/api/v1/reporter", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readResultAfter), len(readResultBefore), "expected to have more reporters after insertion")
}
