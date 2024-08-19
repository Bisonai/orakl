package stats

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
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
)

type websocketId struct {
	Id int32 `db:"id"`
}

func InsertRestCall(ctx context.Context, apiKey string, endpoint string, statusCode int, responseTime time.Duration) error {

	responseTimeMicro := responseTime.Microseconds()
	return db.QueryWithoutResult(ctx, INSERT_REST_CALLS, map[string]any{
		"api_key":       apiKey,
		"endpoint":      endpoint,
		"status_code":   statusCode,
		"response_time": responseTimeMicro,
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

func InsertWebsocketSubscriptions(ctx context.Context, connectionId int32, topics []string) error {
	entries := [][]any{}
	for _, topic := range topics {
		entries = append(entries, []any{connectionId, topic})
	}

	return db.BulkInsert(ctx, "websocket_subscriptions", []string{"connection_id", "topic"}, entries)
}

func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sl := NewStatsLogger(w)
		w.Header()
		defer func() {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				log.Warn().Msg("X-API-Key header is empty")
				return
			}

			endpoint := r.RequestURI
			if endpoint == "/" {
				return
			}

			statusCode := sl.statusCode
			duration := time.Since(start)
			if err := InsertRestCall(r.Context(), key, endpoint, *statusCode, duration); err != nil {
				log.Error().Err(err).Msg("failed to insert rest call")
			}
		}()
		next.ServeHTTP(sl, r)
	})
}

type StatsLogger struct {
	w          *http.ResponseWriter
	body       *bytes.Buffer
	statusCode *int
}

func NewStatsLogger(w http.ResponseWriter) StatsLogger {
	var buf bytes.Buffer
	var statusCode int = 200
	return StatsLogger{
		w:          &w,
		body:       &buf,
		statusCode: &statusCode,
	}
}

func (sl StatsLogger) Write(buf []byte) (int, error) {
	sl.body.Write(buf)
	return (*sl.w).Write(buf)
}

func (sl StatsLogger) Header() http.Header {
	return (*sl.w).Header()

}

func (sl StatsLogger) WriteHeader(statusCode int) {
	(*sl.statusCode) = statusCode
	(*sl.w).WriteHeader(statusCode)
}

func (sl StatsLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := (*sl.w).(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}
