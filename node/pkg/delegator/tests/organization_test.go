//nolint:all
package tests

import (
	"testing"

	"bisonai.com/miko/node/pkg/delegator/organization"
	"bisonai.com/miko/node/pkg/delegator/utils"

	"github.com/stretchr/testify/assert"
)

func TestOrganizationRead(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[[]organization.OrganizationModel](appConfig.App, "/api/v1/organization", nil)
	assert.Nil(t, err)
	assert.Greater(t, len(readResult), 0)
}

func TestOrganizationReadSingle(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[organization.OrganizationModel](appConfig.App, "/api/v1/organization/"+insertedMockOrganization.OrganizationId.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.Name, insertedMockOrganization.Name)
}

func TestOrganizationInsert(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockOrganization1 := organization.OrganizationInsertModel{
		Name: "test2",
	}

	readResultBefore, err := utils.GetRequest[[]organization.OrganizationModel](appConfig.App, "/api/v1/organization", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[organization.OrganizationModel](appConfig.App, "/api/v1/organization", mockOrganization1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Name, mockOrganization1.Name)

	readResultAfter, err := utils.GetRequest[[]organization.OrganizationModel](appConfig.App, "/api/v1/organization", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more organizations after insertion")

	//cleanup
	utils.QueryRowWithoutFiberCtx[organization.OrganizationModel](appConfig.Postgres, organization.DeleteOrganization, map[string]any{"id": insertResult.OrganizationId})
}

func TestOrganizationUpdate(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	mockOrganization1 := organization.OrganizationInsertModel{
		Name: "test3",
	}

	insertResult, err := utils.PostRequest[organization.OrganizationModel](appConfig.App, "/api/v1/organization", mockOrganization1)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Name, mockOrganization1.Name)

	mockOrganization1.Name = "test4"
	updateResult, err := utils.PatchRequest[organization.OrganizationModel](appConfig.App, "/api/v1/organization/"+insertResult.OrganizationId.String(), mockOrganization1)
	assert.Nil(t, err)
	assert.Equal(t, updateResult.Name, mockOrganization1.Name)

	//cleanup
	utils.QueryRowWithoutFiberCtx[organization.OrganizationModel](appConfig.Postgres, organization.DeleteOrganization, map[string]any{"id": insertResult.OrganizationId})
}

func TestOrganizationDelete(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	insertResult, err := utils.PostRequest[organization.OrganizationModel](appConfig.App, "/api/v1/organization", organization.OrganizationInsertModel{Name: "test5"})
	assert.Nil(t, err)

	readResultBefore, err := utils.GetRequest[[]organization.OrganizationModel](appConfig.App, "/api/v1/organization", nil)
	assert.Nil(t, err)

	_, err = utils.DeleteRequest[organization.OrganizationModel](appConfig.App, "/api/v1/organization/"+insertResult.OrganizationId.String(), nil)
	assert.Nil(t, err)

	readResultAfter, err := utils.GetRequest[[]organization.OrganizationModel](appConfig.App, "/api/v1/organization", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readResultAfter), len(readResultBefore), "expected to have less organizations after deletion")
}
