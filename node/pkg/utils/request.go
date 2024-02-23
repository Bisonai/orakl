package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
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
			log.Info().Err(err).Msg("failed to marshal request body")
			return nil, err
		}
		body = bytes.NewReader(marshalledData)
	}

	url, err := url.Parse(urlEndpoint)
	if err != nil {
		log.Info().Err(err).Msg("failed to parse url")
		return nil, err
	}

	req, err := http.NewRequest(
		method,
		url.String(),
		body,
	)
	if err != nil {
		log.Info().Err(err).Msg("failed to create request")
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
		log.Info().Err(err).Msg("failed to make request")
		return result, err
	}
	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Info().Err(err).Msg("failed to read response body")
		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		log.Info().
			Err(err).
			Str("response", string(resultBody)).
			Msg("failed to unmarshal response body")

		return result, err
	}

	return result, nil
}
