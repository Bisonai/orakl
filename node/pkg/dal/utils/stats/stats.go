package stats

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"bisonai.com/miko/node/pkg/db"
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

const (
	DefaultBulkLogsCopyInterval = 10 * time.Minute
	DefaultBufferSize           = 20000
)

type StatsAppConfig struct {
	BulkLogsCopyInterval time.Duration
	BufferSize           int
}

type StatsOption func(*StatsAppConfig)

func WithBulkLogsCopyInterval(interval time.Duration) StatsOption {
	return func(config *StatsAppConfig) {
		config.BulkLogsCopyInterval = interval
	}
}

func WithBufferSize(size int) StatsOption {
	return func(config *StatsAppConfig) {
		config.BufferSize = size
	}
}

type StatsApp struct {
	BulkLogsCopyInterval time.Duration
	RestEntryBuffer      chan *RestEntry
	Cancel               context.CancelFunc
}

type WebsocketId struct {
	Id int32 `db:"id"`
}

type RestEntry struct {
	ApiKey       string
	Endpoint     string
	StatusCode   int
	ResponseTime time.Duration
}

func NewStatsApp(ctx context.Context, opts ...StatsOption) *StatsApp {
	_, cancel := context.WithCancel(ctx)

	config := &StatsAppConfig{
		BulkLogsCopyInterval: DefaultBulkLogsCopyInterval,
		BufferSize:           DefaultBufferSize,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &StatsApp{
		BulkLogsCopyInterval: config.BulkLogsCopyInterval,
		RestEntryBuffer:      make(chan *RestEntry, config.BufferSize),
		Cancel:               cancel,
	}
}

func Start(ctx context.Context) *StatsApp {
	app := NewStatsApp(ctx)
	go app.Run(ctx)
	return app
}

func (a *StatsApp) Stop() {
	a.Cancel()
}

func (a *StatsApp) Run(ctx context.Context) {
	ticker := time.NewTicker(a.BulkLogsCopyInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bulkCopyEntries := [][]any{}
		loop:
			for {
				select {
				case entry := <-a.RestEntryBuffer:
					bulkCopyEntries = append(bulkCopyEntries, []any{entry.ApiKey, entry.Endpoint, entry.StatusCode, entry.ResponseTime.Microseconds()})
				default:
					break loop
				}
			}

			if len(bulkCopyEntries) > 0 {
				_, err := db.BulkCopy(ctx, "rest_calls", []string{"api_key", "endpoint", "status_code", "response_time"}, bulkCopyEntries)
				if err != nil {
					log.Error().Err(err).Msg("failed to bulk copy rest calls")
				}
			}
		}
	}
}

func (a *StatsApp) RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sl := NewStatsLogger(w)
		w.Header()
		defer func() {
			endpoint := r.RequestURI
			if endpoint == "/" {
				return
			}

			key := r.Header.Get("X-API-Key")
			if key == "" {
				log.Warn().Str("endpoint", r.RequestURI).Msg("X-API-Key header is empty")
				return
			}

			statusCode := sl.statusCode
			responseTime := time.Since(start)

			a.RestEntryBuffer <- &RestEntry{
				ApiKey:       key,
				Endpoint:     endpoint,
				StatusCode:   *statusCode,
				ResponseTime: responseTime,
			}
		}()
		next.ServeHTTP(sl, r)
	})
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
	result, err := db.QueryRow[WebsocketId](ctx, INSERT_WEBSOCKET_CONNECTIONS, map[string]any{
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
