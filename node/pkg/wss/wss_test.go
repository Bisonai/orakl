package wss

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnections(t *testing.T) {
	time.Sleep(time.Second)

	// Create a mock WebSocket connection
	conn, err := NewWebsocketHelper(context.Background(), WithEndpoint("ws://localhost:8080/ws"))
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
