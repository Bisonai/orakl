package tests

import (
	"log"
	"os"
	"testing"

	"bisonai.com/orakl/node/pkg/delegator/utils"
	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("env file is not found, continuing without .env file")
	}

	rpk := os.Getenv("TEST_DELEGATOR_REPORTER_PK")

	_testReporterPublicKey, err := utils.GetPublicKey(rpk)
	if err != nil {
		panic(err)
	}
	testReporterPublicKey = _testReporterPublicKey

	_chainId, err := utils.GetChainId()
	if err != nil {
		panic(err)
	}
	loadedChainId = _chainId

	_mockTx, err := makeMockTransaction()
	if err != nil {
		panic(err)
	}
	mockTx = _mockTx

	_mockTxPayload, err := MakeMockTxPayload(mockTx)
	if err != nil {
		panic(err)
	}
	mockTxPayload = _mockTxPayload

	exitVal := m.Run()
	os.Exit(exitVal)
}
