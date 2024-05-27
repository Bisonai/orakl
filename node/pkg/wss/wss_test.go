package wss

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func mockHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to accept websocket connection")
		return
	}
	defer conn.Close(websocket.StatusInternalError, "the sky is falling")

	for {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		err := wsjson.Write(ctx, conn, "Hello")
		if err != nil {
			log.Printf("failed to write message: %v", err)
			conn.Close(websocket.StatusInternalError, "failed to write message")
			return
		}
	}
}

func TestConnections(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", mockHandler)
	// Create an http.Server object
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	// Start the server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()
	defer func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			// unexpected error
			t.Fatalf("Server Shutdown: %v", err)
		}
	}()

	time.Sleep(time.Second)

	// Create a mock WebSocket connection
	conn, err := NewConnection(context.Background(), WithEndpoint("ws://localhost:8080/ws"))
	if err != nil {
		t.Error(err)
	}

	// Test UpdateConnection
	err = UpdateConnection(context.Background(), "testKey", conn)
	assert.NoError(t, err)

	// Test GetConnection
	retrievedConn, err := GetConnection("testKey")
	assert.NoError(t, err)
	assert.Equal(t, conn, retrievedConn)

	// Test RemoveConnection
	err = RemoveConnection("testKey")
	assert.NoError(t, err)

	// Test GetConnection after removal
	retrievedConn, err = GetConnection("testKey")
	assert.Error(t, err)
	assert.Nil(t, retrievedConn)
}
