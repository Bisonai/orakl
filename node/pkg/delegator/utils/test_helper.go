package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/klaytn/klaytn"
	"github.com/klaytn/klaytn/blockchain/types"
	"github.com/klaytn/klaytn/client"
	"github.com/klaytn/klaytn/common"
)

func RawReq(app *fiber.App, method string, endpoint string, requestBody interface{}) error {
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			fmt.Println("failed to marshal request body")
			return err
		}
		body = bytes.NewReader(marshalledData)
	}

	req, err := http.NewRequest(
		method,
		endpoint,
		body,
	)

	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("failed to create request")
		return err
	}
	_, err = app.Test(req, -1)
	if err != nil {
		fmt.Println("failed to call test")
		fmt.Println(err)
		return err
	}
	return nil
}

func req[T any](app *fiber.App, method string, endpoint string, requestBody interface{}) (T, error) {
	var result T
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			fmt.Println("failed to marshal request body")
			return result, err
		}
		body = bytes.NewReader(marshalledData)
	}

	req, err := http.NewRequest(
		method,
		endpoint,
		body,
	)

	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("failed to create request")
		return result, err
	}
	res, err := app.Test(req, -1)
	if err != nil {
		fmt.Println("failed to call test")
		fmt.Println(err)
		return result, err
	}

	resultBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("failed to read result body:" + string(resultBody))
		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		fmt.Println("failed Unmarshal result body:" + string(resultBody))
		return result, err
	}

	return result, nil
}

func GetRequest[T any](app *fiber.App, endpoint string, requestBody interface{}) (T, error) {
	return req[T](app, "GET", endpoint, requestBody)
}

func PostRequest[T any](app *fiber.App, endpoint string, requestBody interface{}) (T, error) {
	return req[T](app, "POST", endpoint, requestBody)
}

func PatchRequest[T any](app *fiber.App, endpoint string, requestBody interface{}) (T, error) {
	return req[T](app, "PATCH", endpoint, requestBody)
}

func DeleteRequest[T any](app *fiber.App, endpoint string, requestBody interface{}) (T, error) {
	return req[T](app, "DELETE", endpoint, requestBody)
}

func UrlRequest[T any](urlEndpoint string, method string, requestBody interface{}) (T, error) {
	var result T
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			fmt.Println("failed to marshal request body")
			return result, err
		}
		body = bytes.NewReader(marshalledData)
	}

	url, err := url.Parse(urlEndpoint)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return result, err
	}

	req, err := http.NewRequest(
		method,
		url.String(),
		body,
	)

	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("failed to create request")
		return result, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(url.String())
		fmt.Println("Error making POST request:", err)
		return result, err
	}
	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		fmt.Println("failed Unmarshal result body:" + string(resultBody))
		return result, err
	}

	return result, nil
}

func DeepCopyMap(src map[string]interface{}) (map[string]interface{}, error) {
	srcJSON, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	dst := make(map[string]interface{})

	err = json.Unmarshal(srcJSON, &dst)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func GetNonce(pk common.Address) (uint64, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return 0, err
	}

	nonce, err := client.PendingNonceAt(context.Background(), pk)
	if err != nil {
		return 0, err
	}

	return nonce, nil
}

func GetGasPrice() (*big.Int, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}

	return gasPrice, nil
}

func GetGasLimit(msg klaytn.CallMsg) (uint64, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return 0, err
	}

	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return 0, err
	}

	return gasLimit, nil
}

func GetChainId() (*big.Int, error) {
	client, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return nil, err
	}

	chainId, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	return chainId, nil
}

func SendTx(signedTx *types.Transaction) error {
	c, err := client.Dial(os.Getenv("PROVIDER_URL"))
	if err != nil {
		return err
	}
	err = c.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	return nil
}
