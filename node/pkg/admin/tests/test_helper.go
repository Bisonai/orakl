package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/bus"
	"github.com/rs/zerolog/log"

	"github.com/gofiber/fiber/v2"
)

func req[T any](app *fiber.App, method string, endpoint string, requestBody interface{}) (T, error) {
	var result T

	resultBody, err := rawReq(app, method, endpoint, requestBody)
	if err != nil {
		log.Error().Err(err).Msg("failed to raw request")
		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		log.Error().Err(err).Msg("failed Unmarshal result body:" + string(resultBody))
		return result, err
	}

	return result, nil
}

func rawReq(app *fiber.App, method string, endpoint string, requestBody interface{}) ([]byte, error) {
	var result []byte
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal request body")
			return result, err
		}
		body = bytes.NewReader(marshalledData)
	}

	req, err := http.NewRequest(
		method,
		endpoint,
		body,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create request")
		return result, err
	}

	req.Header.Set("Content-Type", "application/json")

	apiKey := os.Getenv("API_KEY")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	res, err := app.Test(req, -1)
	if err != nil {
		log.Error().Err(err).Msg("failed to call test")
		return result, err
	}

	resultBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("resultBody", string(resultBody)).Msg("failed to read response body")
		return result, err
	}

	return resultBody, nil
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

func RawPostRequest(app *fiber.App, endpoint string, requestBody interface{}) ([]byte, error) {
	return rawReq(app, "POST", endpoint, requestBody)
}

func UrlRequest[T any](urlEndpoint string, method string, requestBody interface{}) (T, error) {
	var result T
	var body io.Reader

	if requestBody != nil {
		marshalledData, err := json.Marshal(requestBody)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal request body")
			return result, err
		}
		body = bytes.NewReader(marshalledData)
	}

	url, err := url.Parse(urlEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse url")
		return result, err
	}

	req, err := http.NewRequest(
		method,
		url.String(),
		body,
	)

	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Error().Err(err).Msg("failed to create request")
		return result, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", url.String()).Msg("failed to make request")
		return result, err
	}
	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read response body")
		return result, err
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		log.Error().Err(err).Str("resultBody", string(resultBody)).Msg("failed Unmarshal result body")

		return result, err
	}

	return result, nil
}

func waitForMessage(t *testing.T, channel <-chan bus.Message, from, to, command string) {
	go func() {
		select {
		case msg := <-channel:
			if msg.From != from || msg.To != to || msg.Content.Command != command {
				t.Errorf("unexpected message: %v", msg)
			}
			msg.Response <- bus.MessageResponse{Success: true}
		case <-time.After(5 * time.Second):
			t.Errorf("no message received on channel")
		}
	}()
}
