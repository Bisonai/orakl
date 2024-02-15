package tests

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"testing"

	"bisonai.com/orakl/go-delegator/sign"
	"bisonai.com/orakl/go-delegator/utils"

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
	assert.Equal(t, readResult.PrivateKey, "0x12345")

	err = utils.RawReq(appConfig.App, "GET", "/api/v1/sign/initialize", nil)
	assert.Nil(t, err)

	readResultRefreshed, err := utils.GetRequest[utils.FeePayer](appConfig.App, "/readpk", nil)
	assert.Nil(t, err)
	assert.NotEqual(t, readResultRefreshed.PrivateKey, "0x12345")
}

func TestInsert(t *testing.T) {
	t.Skip()
	err := setup()
	assert.Nil(t, err)
	defer t.Cleanup(cleanup)
	defer appConfig.App.Shutdown()

	config := utils.LoadEnvVars()

	// phase 0: test insert

	_mockTx, err := makeMockTransaction()
	if err != nil {
		t.Fatal(err)
	}
	_mockPayload, err := MakeMockTxPayload(_mockTx)
	if err != nil {
		t.Fatal(err)
	}

	readResultBefore, err := utils.GetRequest[[]sign.SignModel](appConfig.App, "/api/v1/sign", nil)
	assert.Nil(t, err)

	insertResult, err := utils.PostRequest[sign.SignModel](appConfig.App, "/api/v1/sign", _mockPayload)
	assert.Nil(t, err)
	assert.Equal(t, insertResult.Nonce, _mockPayload.Nonce)

	readResultAfter, err := utils.GetRequest[[]sign.SignModel](appConfig.App, "/api/v1/sign", nil)
	assert.Nil(t, err)

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more transactions after insertion")

	// vrs, _ := rlp.EncodeToBytes(_mockTx.RawSignatureValues()[0])
	// fmt.Println("vrs: " + hex.EncodeToString(vrs))

	// _recoveredPub, err := crypto.SigToPub(hexutil.MustDecode(*insertResult.SignedRawTx), vrs)
	// if err == nil {
	// 	_recoveredAddr := crypto.PubkeyToAddress(*_recoveredPub)
	// 	fmt.Println("recovered public address: " + _recoveredAddr.String())
	// } else {
	// 	fmt.Println("failed to recover public key: " + err.Error())
	// }

	// phase 1: test tx execution
	callName := "COUNTER()"
	encodedCallName := "0x" + hex.EncodeToString(crypto.Keccak256([]byte(callName))[:4])
	providerUrl := config["PROVIDER_URL"].(string)
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
	fmt.Println(signedRawTx.String())

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

/*
1. transaction.Sign(types.LatestSigner(&params.ChainConfig{ChainID: loadedChainId}), reporterPk)
[{"V":"0x7f5","R":"0xc9f38edfef0af6026d86e2dec370a91d001d270b864ec8cdf1dfac8c6c581396","S":"0x2a7d3f09c00834857caa60031df8a6860471c936557c1ebdb20734771e21e854"}]

2. signedTx, err := types.SignTx(transaction, types.LatestSigner(&params.ChainConfig{ChainID: loadedChainId}), reporterPk)
[{"V":"0x7f5","R":"0xc9f38edfef0af6026d86e2dec370a91d001d270b864ec8cdf1dfac8c6c581396","S":"0x2a7d3f09c00834857caa60031df8a6860471c936557c1ebdb20734771e21e854"}]

rawTx
0x31f89a80850ba43b740083015f909493120927379723583c7a0dd2236fcb255e96949f8094865ab48a9e1f62c6603e696db80e5cb41a20232984d09de08af847f8458207f5a0d1a216850ecd7fe5b52369f94c02860da14d75bb14190d5a0524e4d2fc10942da04284ce1b549b156d314f081833ca723e85e8b8b1c3d67863d5da3540a181465a940026de34522627c5da2b6a5618147a9153c1243ac0

signedRawTx
0x31f8e3808605000000000083015f909493120927379723583c7a0dd2236fcb255e96949f8094865ab48a9e1f62c6603e696db80e5cb41a20232984d09de08af847f8458207f5a0d1a216850ecd7fe5b52369f94c02860da14d75bb14190d5a0524e4d2fc10942da04284ce1b549b156d314f081833ca723e85e8b8b1c3d67863d5da3540a181465a940026de34522627c5da2b6a5618147a9153c1243af847f8458207f5a01964b670512a4f8f5684aae4d02bd0bf8515e54f14e00015247d0979c515065ea025d3f2c1d00bc8e2e54b257ac58bc01522f8286ef49d55d42db79cfa6f705717
*/
