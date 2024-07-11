package stats

import (
	"context"
	"time"

	"bisonai.com/orakl/node/pkg/db"
)

const (
	INSERT_REST_CALLS = `
		INSERT INTO
			rest_calls (api_key, endpoint, status_code, response_time)
		VALUES (@api_key, @endpoint, @status_code, @response_time);`

	INSERT_WEBSOCKET_CONNECTIONS = `
		INSERT INTO
			websocket_connections (api_key)
		VALUES (@api_key) RETURNING id;
	`
	UPDATE_WEBSOCKET_CONNECTIONS = `
		UPDATE websocket_connections
		SET connection_end = NOW(), duration = EXTRACT(EPOCH FROM (NOW() - timestamp)) * 1000
		WHERE id = @id;
	`

	INSERT_WEBSOCKET_SUBSCRIPTIONS = `
		INSERT INTO
			websocket_subscriptions (connection_id, topic)
		VALUES (@connection_id, @topic);
	`
)

type websocketId struct {
	Id int32 `db:"id"`
}

func InsertRestCall(ctx context.Context, apiKey string, endpoint string, statusCode int, responseTime time.Duration) error {
	responseTimeMilli := int(responseTime.Milliseconds())
	return db.QueryWithoutResult(ctx, INSERT_REST_CALLS, map[string]any{
		"api_key":       apiKey,
		"endpoint":      endpoint,
		"status_code":   statusCode,
		"response_time": responseTimeMilli,
	})
}

func InsertWebsocketConnection(ctx context.Context, apiKey string) (int32, error) {
	result, err := db.QueryRow[websocketId](ctx, INSERT_WEBSOCKET_CONNECTIONS, map[string]any{
		"api_key": apiKey,
	})
	if err != nil {
		return 0, err
	}
	return result.Id, nil
}

func UpdateWebsocketConnection(ctx context.Context, connectionId int32) error {
	return db.QueryWithoutResult(ctx, UPDATE_WEBSOCKET_CONNECTIONS, map[string]any{
		"id": connectionId,
	})
}

func InsertWebsocketSubscription(ctx context.Context, connectionId int32, topic string) error {
	return db.QueryWithoutResult(ctx, INSERT_WEBSOCKET_SUBSCRIPTIONS, map[string]any{
		"connection_id": connectionId,
		"topic":         topic,
	})
}
