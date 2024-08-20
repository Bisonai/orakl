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

type mockWebsocketHelper struct {
	*WebsocketHelper
	dialCount int
}

func (m *mockWebsocketHelper) Dial(ctx context.Context) error {
	m.dialCount++
	return nil
}

func (m *mockWebsocketHelper) Write(ctx context.Context, message interface{}) error {
	return nil
}

func (m *mockWebsocketHelper) Close() error {
	m.Conn = nil
	return nil
}

func (m *mockWebsocketHelper) dialAndSubscribe(ctx context.Context) error {
	m.dialCount++
	return nil
}

func TestReconnectTicker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	time.Sleep(time.Second)

	wsHelper := &mockWebsocketHelper{
		WebsocketHelper: &WebsocketHelper{
			Endpoint:          wsURL,
			ReconnectInterval: 100 * time.Millisecond, // Short interval for testing
			InactivityTimeout: 500 * time.Millisecond,
		},
	}

	go wsHelper.Run(ctx, func(context.Context, map[string]any) error {
		return nil
	})

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, 1, wsHelper.dialCount, "Expected initial dial")

	time.Sleep(150 * time.Millisecond) // Wait for reconnect
	assert.Equal(t, 2, wsHelper.dialCount, "Expected reconnect after interval")
}

func TestInactivityTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	time.Sleep(time.Second)

	wsHelper := &mockWebsocketHelper{
		WebsocketHelper: &WebsocketHelper{
			Endpoint:          wsURL,
			ReconnectInterval: 500 * time.Millisecond,
			InactivityTimeout: 100 * time.Millisecond, // Short timeout for testing
		},
	}

	go wsHelper.Run(ctx, func(context.Context, map[string]any) error {
		return nil
	})

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 1, wsHelper.dialCount, "Expected initial dial due to inactivity")
}

func TestResetInactivityOnMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(echoHandler))
	defer server.Close()
	wsURL := "ws" + server.URL[len("http"):] + "/ws"

	wsHelper := &mockWebsocketHelper{
		WebsocketHelper: &WebsocketHelper{
			Endpoint:          wsURL,
			ReconnectInterval: 500 * time.Millisecond,
			InactivityTimeout: 100 * time.Millisecond,
		},
	}

	// Simulate incoming messages to reset inactivity
	go wsHelper.Run(ctx, func(context.Context, map[string]any) error {
		return nil
	})

	time.Sleep(50 * time.Millisecond)
	wsHelper.lastMessageTime = time.Now() // Simulate message received

	time.Sleep(50 * time.Millisecond) // Wait to ensure timer resets
	assert.Equal(t, 0, wsHelper.dialCount, "Expected no dial due to message activity")
}
