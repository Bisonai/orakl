package tests

import (
	"fmt"
	"testing"

	"bisonai.com/miko/node/pkg/api/chain"
	"bisonai.com/miko/node/pkg/api/listener"
	"bisonai.com/miko/node/pkg/api/service"
	"bisonai.com/miko/node/pkg/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestListener(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	var insertChain = chain.ChainInsertModel{Name: "listener-test-chain"}
	var InsertService = service.ServiceInsertModel{Name: "listener-test-service"}

	var insertData = listener.ListenerInsertModel{
		Address:   "0xa",
		EventName: "new_round(uint, uint80)",
		Chain:     "listener-test-chain",
		Service:   "listener-test-service",
	}

	var updateData = listener.ListenerUpdateModel{
		Address:   "0x1",
		EventName: "new_round_v2(uint, uint80)",
	}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")

	chain.Routes(v1)
	service.Routes(v1)
	listener.Routes(v1)

	// insert chain and service before test
	chainInsertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertChain)
	assert.Nil(t, err)
	serviceInsertResult, err := utils.PostRequest[service.ServiceModel](app, "/api/v1/service", InsertService)
	assert.Nil(t, err)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]listener.ListenerModel](app, "/api/v1/listener", map[string]any{"chain": "listener-test-chain", "service": "listener-test-service"})
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	// insert
	insertResult, err := utils.PostRequest[listener.ListenerModel](app, "/api/v1/listener", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]listener.ListenerModel](app, "/api/v1/listener", map[string]any{"chain": "listener-test-chain", "service": "listener-test-service"})
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[listener.ListenerModel](app, "/api/v1/listener/"+insertResult.ListenerId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should get inserted reporter")

	// patch
	patchResult, err := utils.PatchRequest[listener.ListenerModel](app, "/api/v1/listener/"+insertResult.ListenerId.String(), updateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[listener.ListenerModel](app, "/api/v1/listener/"+insertResult.ListenerId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, patchResult, "should be patched")

	// delete
	deleteResult, err := utils.DeleteRequest[listener.ListenerModel](app, "/api/v1/listener/"+insertResult.ListenerId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	// read all after delete
	readAllResultAfterDeletion, err := utils.GetRequest[[]listener.ListenerModel](app, "/api/v1/listener", map[string]any{"chain": "listener-test-chain", "service": "listener-test-service"})
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)

	// delete chain and service (cleanup)
	_, err = utils.DeleteRequest[chain.ChainModel](app, "/api/v1/chain/"+chainInsertResult.ChainId.String(), nil)
	assert.Nil(t, err)
	_, err = utils.DeleteRequest[service.ServiceModel](app, "/api/v1/service/"+serviceInsertResult.ServiceId.String(), nil)
	assert.Nil(t, err)
}
