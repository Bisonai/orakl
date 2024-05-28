//nolint:all
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
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", echoHandler)
	// Create an http.Server object
	srv := &http.Server{
		Addr:    ":8088",
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
		if err := srv.Close(); err != nil {
			// unexpected error
			t.Fatalf("Server Shutdown: %v", err)
		}
	}()

	time.Sleep(time.Second)

	// Create a WebSocket connection
	conn, err := NewWebsocketHelper(context.Background(), WithEndpoint("ws://localhost:8088/ws"))
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
