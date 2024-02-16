package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func GetRequest[T any](urlEndpoint string, requestBody interface{}, headers map[string]string) (T, error) {
	return UrlRequest[T](urlEndpoint, "GET", requestBody, headers)
}

func GetRequestRaw(urlEndpoint string, requestBody interface{}, headers map[string]string) (*http.Response, error) {
	return UrlRequestRaw(urlEndpoint, "GET", requestBody, headers)
}

func UrlRequestRaw(urlEndpoint string, method string, requestBody interface{}, headers map[string]string) (*http.Response, error) {
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			fmt.Println("failed to marshal request body")
			return nil, err
		}
		body = bytes.NewReader(marshalledData)
	}

	url, err := url.Parse(urlEndpoint)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return nil, err
	}

	req, err := http.NewRequest(
		method,
		url.String(),
		body,
	)
	if err != nil {
		fmt.Println("failed to create request")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{
		Timeout: time.Second, // Set the timeout to 1 second
	}

	return client.Do(req)
}

func UrlRequest[T any](urlEndpoint string, method string, requestBody interface{}, headers map[string]string) (T, error) {
	var result T
	response, err := UrlRequestRaw(urlEndpoint, method, requestBody, headers)
	if err != nil {
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
