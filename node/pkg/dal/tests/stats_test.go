//nolint:all
package test

import (
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/dal/utils/stats"
	"bisonai.com/miko/node/pkg/db"

	"github.com/stretchr/testify/assert"
)

type RestCall struct {
	ID           int32     `db:"id"`
	ApiKey       string    `db:"api_key"`
	Endpoint     string    `db:"endpoint"`
	Timestamp    time.Time `db:"timestamp"`
	StatusCode   int       `db:"status_code"`
	ResponseTime int       `db:"response_time"`
}

type WebsocketConnection struct {
	ID            int32      `db:"id"`
	ApiKey        string     `db:"api_key"`
	Timestamp     time.Time  `db:"timestamp"`
	ConnectionEnd *time.Time `db:"connection_end"`
	Duration      *int       `db:"duration"`
}

type WebsocketSubscription struct {
	ID           int32     `db:"id"`
	ConnectionId int32     `db:"connection_id"`
	Topic        string    `db:"topic"`
	Timestamp    time.Time `db:"timestamp"`
}

func TestInsertRestCall(t *testing.T) {
	ctx := context.Background()
	_ = db.QueryWithoutResult(ctx, "DELETE FROM rest_calls", nil)

	err := stats.InsertRestCall(ctx, "testApiKey", "test", 200, 10*time.Millisecond)
	assert.NoError(t, err)

	result, err := db.QueryRows[RestCall](ctx, "SELECT * FROM rest_calls", nil)
	assert.NoError(t, err)
	assert.Greater(t, len(result), 0)

	assert.Equal(t, "testApiKey", result[0].ApiKey)
	err = db.QueryWithoutResult(ctx, "DELETE FROM rest_calls", nil)
	assert.NoError(t, err)
}

func TestInsertWebsocketConnection(t *testing.T) {
	ctx := context.Background()
	_ = db.QueryWithoutResult(ctx, "DELETE FROM websocket_connections", nil)

	id, err := stats.InsertWebsocketConnection(ctx, "testApiKey")
	assert.NoError(t, err)
	assert.Greater(t, id, int32(0))

	result, err := db.QueryRows[WebsocketConnection](ctx, "SELECT * FROM websocket_connections", nil)
	assert.NoError(t, err)
	assert.Greater(t, len(result), 0)

	assert.Equal(t, "testApiKey", result[0].ApiKey)
	err = db.QueryWithoutResult(ctx, "DELETE FROM websocket_connections", nil)
	assert.NoError(t, err)
}

func TestUpdateWebsocketConnection(t *testing.T) {
	ctx := context.Background()
	_ = db.QueryWithoutResult(ctx, "DELETE FROM websocket_connections", nil)
	id, err := stats.InsertWebsocketConnection(ctx, "testApiKey")
	assert.NoError(t, err)
	assert.Greater(t, id, int32(0))

	err = stats.UpdateWebsocketConnection(ctx, id)
	assert.NoError(t, err)

	result, err := db.QueryRows[WebsocketConnection](ctx, "SELECT * FROM websocket_connections", nil)
	assert.NoError(t, err)
	assert.Greater(t, len(result), 0)
	assert.Equal(t, "testApiKey", result[0].ApiKey)
	assert.NotEqual(t, 0, result[0].Duration)
	err = db.QueryWithoutResult(ctx, "DELETE FROM websocket_connections", nil)
	assert.NoError(t, err)
}

func TestWebsocketSubcription(t *testing.T) {
	ctx := context.Background()
	_ = db.QueryWithoutResult(ctx, "DELETE FROM websocket_subscriptions", nil)
	_ = db.QueryWithoutResult(ctx, "DELETE FROM websocket_connections", nil)
	id, err := stats.InsertWebsocketConnection(ctx, "testApiKey")
	assert.NoError(t, err)
	assert.Greater(t, id, int32(0))

	err = stats.InsertWebsocketSubscriptions(ctx, id, []string{"test_topic"})
	assert.NoError(t, err)

	result, err := db.QueryRows[WebsocketSubscription](ctx, "SELECT * FROM websocket_subscriptions", nil)
	assert.NoError(t, err)
	assert.Greater(t, len(result), 0)
	assert.Equal(t, "test_topic", result[0].Topic)
	err = db.QueryWithoutResult(ctx, "DELETE FROM websocket_subscriptions", nil)
	assert.NoError(t, err)
	err = db.QueryWithoutResult(ctx, "DELETE FROM websocket_connections", nil)
	assert.NoError(t, err)
}
