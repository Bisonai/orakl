package tests

import (
	"fmt"

	"testing"
	"time"

	"bisonai.com/orakl/api/apierr"
	"bisonai.com/orakl/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestApiErr(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	now := utils.CustomDateTime{Time: time.Now()}
	insertData := apierr.ErrorInsertModel{
		RequestId: "66649924661314489704239946349158829048302840686075232939396730072454733114998",
		Timestamp: &now,
		Code:      "10020",
		Name:      "MissingKeyInJson",
		Stack: `MissingKeyInJson
		at wrapper (file:///app/dist/worker/reducer.js:19:23)
		at file:///app/dist/utils.js:11:61
		at Array.reduce (<anonymous>)
		at file:///app/dist/utils.js:11:44
		at processRequest (file:///app/dist/worker/request-response.js:58:34)
		at process.processTicksAndRejections (node:internal/process/task_queues:95:5)
		at async Worker.wrapper [as processFn] (file:///app/dist/worker/request-response.js:27:25)
		at async Worker.processJob (/app/node_modules/bullmq/dist/cjs/classes/worker.js:339:28)
		at async Worker.retryIfFailed (/app/node_modules/bullmq/dist/cjs/classes/worker.js:513:24)`,
	}
	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")
	apierr.Routes(v1)

	// read all before insertion
	readAllResultBefore, err := utils.GetRequest[[]apierr.ErrorModel](app, "/api/v1/error", nil)
	assert.Nil(t, err)
	totalBefore := len(readAllResultBefore)

	// insert
	insertResult, err := utils.PostRequest[apierr.ErrorModel](app, "/api/v1/error", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]apierr.ErrorModel](app, "/api/v1/error", nil)
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[apierr.ErrorModel](app, "/api/v1/error/"+insertResult.ERROR_ID.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should get inserted service")

	// delete
	deleteResult, err := utils.DeleteRequest[apierr.ErrorModel](app, "/api/v1/error/"+insertResult.ERROR_ID.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, deleteResult, "should be deleted")

	readAllResultAfterDeletion, err := utils.GetRequest[[]apierr.ErrorModel](app, "/api/v1/error", nil)
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)
}
