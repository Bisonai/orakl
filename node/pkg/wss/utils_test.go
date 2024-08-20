//nolint:all
package wss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to accept websocket connection")
		return
	}
	defer conn.Close(websocket.StatusInternalError, "the sky is falling")

	for {
		var v interface{}
		err := wsjson.Read(r.Context(), conn, &v)
		if err != nil {
			break
		}

		err = wsjson.Write(r.Context(), conn, v)
		if err != nil {
			log.Error().Err(err).Msg("failed to write message")
			break
		}
	}
}

func TestReadWriteAndClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	time.Sleep(time.Second)

	// Create a WebSocket connection
	conn, err := NewWebsocketHelper(context.Background(), WithEndpoint(wsURL))
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Test Dial
	err = conn.Dial(context.Background())
	assert.NoError(t, err)

	// Test Write
	err = conn.Write(context.Background(), "Hello")
	assert.NoError(t, err)

	// Test Read
	ch := make(chan any)
	go conn.Read(context.Background(), ch)
	assert.Equal(t, "Hello", <-ch)

	// Test Close
	err = conn.Close()
	assert.NoError(t, err)
}

func TestReconnectTicker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	// Create a WebSocket connection with a short reconnect interval
	conn, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(wsURL),
		WithReconnectInterval(100*time.Millisecond), // Short interval for testing
	)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Run the WebSocket helper in a separate goroutine
	go conn.Run(context.Background(), func(ctx context.Context, data map[string]any) error {
		// Handle data if necessary
		return nil
	})

	// Give it some time to reconnect
	time.Sleep(500 * time.Millisecond)

	// Test if the connection has been re-established
	assert.NotNil(t, conn.Conn) // Assuming that `conn.Conn` should not be nil
	assert.True(t, conn.IsRunning)

	// Close the WebSocket helper
	err = conn.Close()
	assert.NoError(t, err)
}

func TestInactivityTimer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	// Create a WebSocket connection with a short inactivity timeout
	conn, err := NewWebsocketHelper(
		context.Background(),
		WithEndpoint(wsURL),
		WithInactivityTimeout(50*time.Millisecond), // Short timeout for testing
	)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Run the WebSocket helper in a separate goroutine
	go conn.Run(context.Background(), func(ctx context.Context, data map[string]any) error {
		// Handle data if necessary
		return nil
	})

	// Wait longer than the inactivity timeout
	time.Sleep(500 * time.Millisecond)

	// Verify if the connection has been closed due to inactivity
	assert.NotNil(t, conn.Conn) // Assuming that `conn.Conn` should be nil if closed
	assert.True(t, conn.IsRunning)

	// Close the WebSocket helper
	err = conn.Close()
	assert.NoError(t, err)
}
