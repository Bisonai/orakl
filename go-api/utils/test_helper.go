package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

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
