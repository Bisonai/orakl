//nolint:all
package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"bisonai.com/orakl/node/pkg/utils/request"
)

type TestResponse struct {
	Message string `json:"message"`
}

type TestRequestBody struct {
	Test string `json:"test"`
}

func createMockProxyServer() *httptest.Server {
	// Create a mock HTTP server that acts as a proxy
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new request to the URL specified in the original request
		req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Copy the headers from the original request to the new request
		for name, values := range r.Header {
			req.Header[name] = values
		}

		// Send the request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy the response to the response writer
		io.Copy(w, resp.Body)
	}))
	return proxy
}

func createMockServer() *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request body is present
		var requestBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check if the header is present
		if r.Header.Get("Test-Header") != "test-value" {
			http.Error(w, "Test-Header not found", http.StatusBadRequest)
			return
		}

		if requestBody["test"] != "value" {
			http.Error(w, "Request body not found", http.StatusBadRequest)
			return
		}

		response := map[string]string{
			"message": "Mock server response",
		}

		// Set Content-Type header so that clients will know how to read the response
		w.Header().Set("Content-Type", "application/json")

		// Encode the data to JSON and send as the response
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	return server
}

func TestGetRequest(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	requestBody := TestRequestBody{
		Test: "value",
	}

	testResponse, err := request.GetRequest[TestResponse](server.URL, requestBody, headers)
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if testResponse.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", testResponse.Message)
	}
}

func TestGetRequestRaw(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	requestBody := TestRequestBody{
		Test: "value",
	}

	res, err := request.GetRequestRaw(server.URL, requestBody, headers)
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected status code 200 but got %v", res.StatusCode)
	}
	var result TestResponse
	resultBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error("Error reading response body:", err)
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		t.Error("Error unmarshalling response body:", err)
	}

	if result.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", result.Message)
	}
}

func TestUrlRequest(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServer()
	defer server.Close()

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	requestBody := TestRequestBody{
		Test: "value",
	}

	testResponse, err := request.UrlRequest[TestResponse](server.URL, "GET", requestBody, headers, "")
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if testResponse.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", testResponse.Message)
	}
}

func TestUrlRequestRaw(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServer()
	defer server.Close()

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	requestBody := TestRequestBody{
		Test: "value",
	}

	res, err := request.UrlRequestRaw(server.URL, "GET", requestBody, headers, "")
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected status code 200 but got %v", res.StatusCode)
	}

	var result TestResponse
	resultBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error("Error reading response body:", err)
	}

	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		t.Error("Error unmarshalling response body:", err)
	}

	if result.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", result.Message)
	}
}

func TestGetRequestProxy(t *testing.T) {
	// Create a mock HTTP server that acts as a proxy
	proxy := createMockProxyServer()
	// Close the server when test finishes
	defer proxy.Close()

	// Create a mock HTTP server
	server := createMockServer()
	defer server.Close()

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	requestBody := TestRequestBody{
		Test: "value",
	}

	testResponse, err := request.GetRequestProxy[TestResponse](server.URL, requestBody, headers, proxy.URL)
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if testResponse.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", testResponse.Message)
	}
}

func TestUrlRequestProxy(t *testing.T) {
	// Create a mock HTTP server that acts as a proxy
	proxy := createMockProxyServer()
	defer proxy.Close()

	server := createMockServer()
	defer server.Close()

	requestBody := TestRequestBody{
		Test: "value",
	}

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	testResponse, err := request.UrlRequest[TestResponse](server.URL, "GET", requestBody, headers, proxy.URL)
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if testResponse.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", testResponse.Message)
	}
}

func TestUrlRequestRawProxy(t *testing.T) {
	// Create a mock HTTP server that acts as a proxy
	proxy := createMockProxyServer()
	// Close the server when test finishes
	defer proxy.Close()

	server := createMockServer()
	defer server.Close()

	requestBody := TestRequestBody{
		Test: "value",
	}

	headers := map[string]string{
		"Test-Header": "test-value",
	}

	res, err := request.UrlRequestRaw(server.URL, "GET", requestBody, headers, proxy.URL)
	if err != nil {
		t.Errorf("Error making request: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected status code 200 but got %v", res.StatusCode)
	}

	resultBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error("Error reading response body:", err)
	}

	var result TestResponse
	err = json.Unmarshal(resultBody, &result)
	if err != nil {
		t.Error("Error unmarshalling response body:", err)
	}

	if result.Message != "Mock server response" {
		t.Errorf("Expected response message 'Mock server response' but got %v", result.Message)
	}
}
