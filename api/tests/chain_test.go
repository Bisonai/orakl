package tests

import (
	"fmt"
	"testing"

	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestChain(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	var insertData = chain.ChainInsertModel{Name: "cypress"}
	var updateData = chain.ChainInsertModel{Name: "cypress2"}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")
	chain.Routes(v1)

	// read all before insertion
	readAllResultBefore, err := utils.GetRequest[[]chain.ChainModel](app, "/api/v1/chain", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResultBefore)

	// insert
	insertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]chain.ChainModel](app, "/api/v1/chain", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[chain.ChainModel](app, "/api/v1/chain/"+insertResult.ChainId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should get inserted chain")

	// patch
	patchResult, err := utils.PatchRequest[chain.ChainModel](app, "/api/v1/chain/"+insertResult.ChainId.String(), updateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[chain.ChainModel](app, "/api/v1/chain/"+insertResult.ChainId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, patchResult, "should be patched")

	// delete
	deleteResult, err := utils.DeleteRequest[chain.ChainModel](app, "/api/v1/chain/"+insertResult.ChainId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	// read all after delete
	readAllResultAfterDeletion, err := utils.GetRequest[[]chain.ChainModel](app, "/api/v1/chain", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)
}
