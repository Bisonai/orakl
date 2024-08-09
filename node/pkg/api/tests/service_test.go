package tests

import (
	"fmt"
	"testing"

	"bisonai.com/orakl/node/pkg/api/service"
	"bisonai.com/orakl/node/pkg/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestService(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	var insertData = service.ServiceInsertModel{Name: "SERVICE_TEST"}
	var updateData = service.ServiceInsertModel{Name: "SERVICE_TEST_2"}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")
	service.Routes(v1)

	// read all before insertion
	readAllResultBefore, err := utils.GetRequest[[]service.ServiceModel](app, "/api/v1/service", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResultBefore)

	// insert
	insertResult, err := utils.PostRequest[service.ServiceModel](app, "/api/v1/service", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]service.ServiceModel](app, "/api/v1/service", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[service.ServiceModel](app, "/api/v1/service/"+insertResult.ServiceId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should get inserted service")

	// patch
	patchResult, err := utils.PatchRequest[service.ServiceModel](app, "/api/v1/service/"+insertResult.ServiceId.String(), updateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[service.ServiceModel](app, "/api/v1/service/"+insertResult.ServiceId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, patchResult, "should be patched")

	// delete
	deleteResult, err := utils.DeleteRequest[service.ServiceModel](app, "/api/v1/service/"+insertResult.ServiceId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	// read all after delete
	readAllResultAfterDeletion, err := utils.GetRequest[[]service.ServiceModel](app, "/api/v1/service", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)
}
