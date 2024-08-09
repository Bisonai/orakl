package tests

import (
	"fmt"
	"testing"

	"bisonai.com/orakl/node/pkg/api/chain"
	"bisonai.com/orakl/node/pkg/api/reporter"
	"bisonai.com/orakl/node/pkg/api/service"
	"bisonai.com/orakl/node/pkg/api/utils"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestReporter(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Print("env file is not found, continueing without .env file")
	}

	var insertChain = chain.ChainInsertModel{Name: "reporter-test-chain"}
	var insertService = service.ServiceInsertModel{Name: "reporter-test-service"}

	var insertData = reporter.ReporterInsertModel{
		Address:       "0xa",
		PrivateKey:    "0xb",
		OracleAddress: "0xc",
		Chain:         "reporter-test-chain",
		Service:       "reporter-test-service",
	}

	var updateData = reporter.ReporterUpdateModel{
		Address:       "0x1",
		PrivateKey:    "0x2",
		OracleAddress: "0x3",
	}

	appConfig, _ := utils.Setup()

	pgxClient := appConfig.Postgres
	app := appConfig.App

	defer pgxClient.Close()
	v1 := app.Group("/api/v1")

	chain.Routes(v1)
	service.Routes(v1)
	reporter.Routes(v1)

	// insert chain and service before test
	chainInsertResult, err := utils.PostRequest[chain.ChainModel](app, "/api/v1/chain", insertChain)
	assert.Nil(t, err)
	serviceInsertResult, err := utils.PostRequest[service.ServiceModel](app, "/api/v1/service", insertService)
	assert.Nil(t, err)

	// read all before insertion
	readAllResult, err := utils.GetRequest[[]reporter.ReporterModel](app, "/api/v1/reporter", map[string]any{"chain": "reporter-test-chain", "service": "reporter-test-service"})
	assert.Nil(t, err)
	totalBefore := len(readAllResult)

	// insert
	insertResult, err := utils.PostRequest[reporter.ReporterModel](app, "/api/v1/reporter", insertData)
	assert.Nil(t, err)

	// read all after insertion
	readAllResultAfter, err := utils.GetRequest[[]reporter.ReporterModel](app, "/api/v1/reporter", map[string]any{"chain": "reporter-test-chain", "service": "reporter-test-service"})
	assert.Nil(t, err)
	totalAfter := len(readAllResultAfter)
	assert.Less(t, totalBefore, totalAfter)

	// read single
	singleReadResult, err := utils.GetRequest[reporter.ReporterModel](app, "/api/v1/reporter/"+insertResult.ReporterId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, insertResult, singleReadResult, "should be inserted")

	// patch
	patchResult, err := utils.PatchRequest[reporter.ReporterModel](app, "/api/v1/reporter/"+insertResult.ReporterId.String(), updateData)
	assert.Nil(t, err)
	singleReadResult, err = utils.GetRequest[reporter.ReporterModel](app, "/api/v1/reporter/"+insertResult.ReporterId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, singleReadResult, patchResult, "should be patched")

	// delete
	deleteResult, err := utils.DeleteRequest[reporter.ReporterModel](app, "/api/v1/reporter/"+insertResult.ReporterId.String(), nil)
	assert.Nil(t, err)
	assert.Equalf(t, patchResult, deleteResult, "should be deleted")

	// read all after delete
	readAllResultAfterDeletion, err := utils.GetRequest[[]reporter.ReporterModel](app, "/api/v1/reporter", map[string]any{"chain": "reporter-test-chain", "service": "reporter-test-service"})
	assert.Nil(t, err)
	assert.Less(t, len(readAllResultAfterDeletion), totalAfter)

	// delete chain and service (cleanup)
	_, err = utils.DeleteRequest[chain.ChainModel](app, "/api/v1/chain/"+chainInsertResult.ChainId.String(), nil)
	assert.Nil(t, err)
	_, err = utils.DeleteRequest[service.ServiceModel](app, "/api/v1/service/"+serviceInsertResult.ServiceId.String(), nil)
	assert.Nil(t, err)
}
