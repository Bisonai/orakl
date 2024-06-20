package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/rs/zerolog/log"
)

func GetRequest[T any](urlEndpoint string, requestBody interface{}, headers map[string]string) (T, error) {
	return UrlRequest[T](urlEndpoint, "GET", requestBody, headers, "")
}

func GetRequestRaw(urlEndpoint string, requestBody interface{}, headers map[string]string) (*http.Response, error) {
	return UrlRequestRaw(urlEndpoint, "GET", requestBody, headers, "")
}

func UrlRequest[T any](urlEndpoint string, method string, requestBody interface{}, headers map[string]string, proxy string) (T, error) {
	var result T
	response, err := UrlRequestRaw(urlEndpoint, method, requestBody, headers, proxy)
	if err != nil {
		log.Error().Err(err).Msg("failed to make request")
		return result, err
	}

	if response.StatusCode != http.StatusOK {
		log.Info().
			Int("status", response.StatusCode).
			Str("url", urlEndpoint).
			Msg("failed to make request")
		return result, errorSentinel.ErrRequestStatusNotOk
	}

	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read response body")

		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		log.Error().Err(err).Str("resultBody", string(resultBody)).Msg("failed to unmarshal response body")
		return result, err
	}

	return result, nil
}

func UrlRequestRaw(urlEndpoint string, method string, requestBody interface{}, headers map[string]string, proxy string) (*http.Response, error) {
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal request body")
			return nil, err
		}
		body = bytes.NewReader(marshalledData)
	}

	url, err := url.Parse(urlEndpoint)
	if err != nil {
		log.Error().Err(err).Str("url", urlEndpoint).Msg("failed to parse url")
		return nil, err
	}

	req, err := http.NewRequest(
		method,
		url.String(),
		body,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create request")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if len(headers) > 0 {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{
		Timeout: 2 * time.Second, // Set the timeout to 1 second
	}

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			log.Error().Err(err).Str("proxy", proxy).Msg("failed to parse proxy")
			return nil, err
		}

		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}

		if url.Scheme == "https" {
			url.Scheme = "http"
			req.URL = url
		}
	}

	return client.Do(req)
}

func GetRequestProxy[T any](urlEndpoint string, requestBody interface{}, headers map[string]string, proxy string) (T, error) {
	log.Debug().Str("url", urlEndpoint).Str("proxy", proxy).Msg("making request with proxy")
	return UrlRequest[T](urlEndpoint, "GET", requestBody, headers, proxy)
}
