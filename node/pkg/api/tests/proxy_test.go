package tests

import (
	"fmt"
	"testing"

	"bisonai.com/orakl/node/pkg/api/proxy"
	"bisonai.com/orakl/node/pkg/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestProxy(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}
	var location = "kr"
	var portNumber = utils.CustomInt32(5000)
	var insertData = proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     &portNumber,
	}

	var updateData = proxy.ProxyInsertModel{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     &portNumber,
		Location: &location,
	}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")
	proxy.Routes(v1)

	// read all before insertion
	readAllResultBefore, err := utils.GetRequest[[]proxy.ProxyModel](app, "/api/v1/proxy", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResultBefore)

	// insert
	insertResult, err := utils.PostRequest[proxy.ProxyModel](app, "/api/v1/proxy", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]proxy.ProxyModel](app, "/api/v1/proxy", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[proxy.ProxyModel](app, "/api/v1/proxy/"+insertResult.Id.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should get inserted proxy")

	// patch
	patchResult, err := utils.PatchRequest[proxy.ProxyModel](app, "/api/v1/proxy/"+insertResult.Id.String(), updateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[proxy.ProxyModel](app, "/api/v1/proxy/"+insertResult.Id.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, patchResult, "should be patched")

	// delete
	deleteResult, err := utils.DeleteRequest[proxy.ProxyModel](app, "/api/v1/proxy/"+insertResult.Id.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	// read all after delete
	readAllResultAfterDeletion, err := utils.GetRequest[[]proxy.ProxyModel](app, "/api/v1/proxy", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)
}
