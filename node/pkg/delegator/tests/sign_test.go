//nolint:all
package tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"net/url"
	"os"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/delegator/sign"
	"bisonai.com/orakl/node/pkg/delegator/utils"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gofiber/fiber/v2"
	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/accounts/abi/bind"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
	"github.com/klaytn/klaytn/crypto"
	"github.com/klaytn/klaytn/rlp"
	"github.com/stretchr/testify/assert"
)

func TestSignRead(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[[]sign.SignModel](appConfig.App, "/api/v1/sign", nil)
	assert.Nil(t, err)
	assert.Greater(t, len(readResult), 0)
}

func TestSignReadSingle(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	readResult, err := utils.GetRequest[sign.SignModel](appConfig.App, "/api/v1/sign/"+insertedMockTx.Id.String(), nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.Id, insertedMockTx.Id)
}

func TestInitialize(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	params := url.Values{}
	params.Add("feePayerPrivateKey", "0x12345")

	appConfig.App.Get("/readpk", func(c *fiber.Ctx) error {
		fp, error := utils.GetFeePayer(c)
		if error != nil {
			return error
		}
		return c.JSON(utils.FeePayer{PrivateKey: fp})
	})

	//_, err = utils.GetRequest[interface{}](appConfig.App, "/api/v1/sign/initialize?feePayerPrivateKey=0x12345", nil)

	err = utils.RawReq(appConfig.App, "GET", "/api/v1/sign/initialize?feePayerPrivateKey=0x12345", nil)
	assert.Nil(t, err)

	readResult, err := utils.GetRequest[utils.FeePayer](appConfig.App, "/readpk", nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult.PrivateKey, "12345")

	err = utils.RawReq(appConfig.App, "GET", "/api/v1/sign/initialize", nil)
	assert.Nil(t, err)

	readResultRefreshed, err := utils.GetRequest[utils.FeePayer](appConfig.App, "/readpk", nil)
	assert.Nil(t, err)
	assert.NotEqual(t, readResultRefreshed.PrivateKey, "12345")
}

func TestGetFeePayerAddress(t *testing.T) {
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	appConfig.App.Get("/readpk", func(c *fiber.Ctx) error {
		fp, error := utils.GetFeePayer(c)
		if error != nil {
			return error
		}
		return c.JSON(utils.FeePayer{PrivateKey: fp})
	})

	err = utils.RawReq(appConfig.App, "GET", "/api/v1/sign/initialize?feePayerPrivateKey=0x6014d3aa8be8980fd90146d10176e7ef632bdba96279e8bbe55421e79a979a2e", nil)
	assert.Nil(t, err)

	readResult, err := utils.GetRequest[string](appConfig.App, "/api/v1/sign/feePayer", nil)
	assert.Nil(t, err)
	assert.Equal(t, readResult, "0xda2E0E7089a479ef4A75a8c6Cc78426B9270EC08")
}

func TestInsert(t *testing.T) {
	t.Skip()
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	// phase 0: test insert
	_mockTx, err := makeMockTransaction()
	if err != nil {
		t.Fatal(err)
	}
	_mockPayload, err := MakeMockTxPayload(_mockTx)
	if err != nil {
		t.Fatal(err)
	}

	readResultBefore, err := utils.GetRequest[[]sign.SignModel](appConfig.App, "/api/v1/sign/v2", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[sign.SignModel](appConfig.App, "/api/v1/sign/v2", _mockPayload)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Nonce, _mockPayload.Nonce)

	readResultAfter, err := utils.GetRequest[[]sign.SignModel](appConfig.App, "/api/v1/sign/v2", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more transactions after insertion")

	// phase 1: test tx execution
	callName := "COUNTER()"
	encodedCallName := "0x" + hex.EncodeToString(crypto.Keccak256([]byte(callName))[:4])
	providerUrl := os.Getenv("PROVIDER_URL")
	client, err := client.Dial(providerUrl)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// read contract before
	contractAddr := common.HexToAddress(_mockPayload.To)
	callMsg := klaytn.CallMsg{
		To:   &contractAddr,
		Data: common.FromHex(encodedCallName),
	}
	callResultBefore, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		t.Fatal(err)
	}
	beforeNum, err := strconv.ParseInt(common.BytesToHash(callResultBefore).Hex()[2:], 16, 64)
	if err != nil {
		t.Fatal(err)
	}

	signedRawTxBytes := hexutil.MustDecode(*insertResult.SignedRawTx)

	var signedRawTx = new(types.Transaction)
	rlp.Decode(bytes.NewReader(signedRawTxBytes), signedRawTx)

	err = client.SendTransaction(context.Background(), signedRawTx)
	// _, err = client.SendRawTransaction(context.Background(), signedRawTx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bind.WaitMined(context.Background(), client, signedRawTx)
	if err != nil {
		t.Fatal(err)
	}

	// read contract after
	callResultAfter, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		t.Fatal(err)
	}
	afterNum, err := strconv.ParseInt(common.BytesToHash(callResultAfter).Hex()[2:], 16, 64)
	if err != nil {
		t.Fatal(err)
	}

	assert.Greater(t, afterNum, beforeNum)

	//cleanup
	utils.QueryRowWithoutFiberCtx[sign.SignModel](appConfig.Postgres, sign.DeleteTransactionById, map[string]any{"id": insertResult.Id})
}
