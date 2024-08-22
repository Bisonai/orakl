package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/rs/zerolog/log"
)

const DefaultTimeout = 2 * time.Second

type RequestConfig struct {
	Timeout  time.Duration
	Endpoint string
	Body     interface{}
	Headers  map[string]string
	Proxy    string
	Method   string
}

type RequestOption func(*RequestConfig)

func WithTimeout(timeout time.Duration) RequestOption {
	return func(config *RequestConfig) {
		config.Timeout = timeout
	}
}

func WithEndpoint(endpoint string) RequestOption {
	return func(config *RequestConfig) {
		config.Endpoint = endpoint
	}
}

func WithBody(body interface{}) RequestOption {
	return func(config *RequestConfig) {
		config.Body = body
	}
}

func WithHeaders(headers map[string]string) RequestOption {
	return func(config *RequestConfig) {
		config.Headers = headers
	}
}

func WithProxy(proxy string) RequestOption {
	return func(config *RequestConfig) {
		config.Proxy = proxy
	}
}

func WithMethod(method string) RequestOption {
	return func(config *RequestConfig) {
		config.Method = method
	}
}

func Request[T any](opts ...RequestOption) (T, error) {
	var result T

	config := RequestConfig{
		Timeout: DefaultTimeout,
		Method:  "GET",
	}
	for _, opt := range opts {
		opt(&config)
	}
	response, err := requestRaw(config)
	if err != nil {
		log.Error().Err(err).Msg("failed to make request")
		return result, err
	}

	if response.StatusCode != http.StatusOK {
		log.Info().
			Int("status", response.StatusCode).
			Str("url", config.Endpoint).
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

func RequestRaw(opts ...RequestOption) (*http.Response, error) {
	config := RequestConfig{
		Timeout: DefaultTimeout,
		Method:  "GET",
	}
	for _, opt := range opts {
		opt(&config)
	}
	return requestRaw(config)
}

func requestRaw(config RequestConfig) (*http.Response, error) {
	var body io.Reader

	if config.Body != nil {
		marshalledData, err := json.Marshal(config.Body)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal request body")
			return nil, err
		}
		body = bytes.NewReader(marshalledData)
	}

	url, err := url.Parse(config.Endpoint)
	if err != nil {
		log.Error().Err(err).Str("url", config.Endpoint).Msg("failed to parse url")
		return nil, err
	}

	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
	if !validMethods[config.Method] {
		log.Error().Str("method", config.Method).Msg("invalid method")
		return nil, errorSentinel.ErrRequestInvalidMethod
	}

	req, err := http.NewRequest(
		config.Method,
		url.String(),
		body,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create request")
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if len(config.Headers) > 0 {
		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	if config.Proxy != "" {
		err := setProxy(client, config.Proxy)
		if err != nil {
			return nil, err
		}

		if url.Scheme == "https" {
			url.Scheme = "http"
			req.URL = url
		}
	}

	return client.Do(req)
}

func setProxy(client *http.Client, proxyURL string) error {
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Error().Err(err).Str("proxy", proxyURL).Msg("failed to parse proxy")
		return err
	}

	client.Transport = &http.Transport{
		Proxy: http.ProxyURL(parsedURL),
	}

	return nil
}
