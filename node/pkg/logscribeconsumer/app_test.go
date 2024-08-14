//nolint:all
package logscribeconsumer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type LogEntry struct {
	Level     zerolog.Level   `json:"level" db:"level"`
	Message   string          `json:"message" db:"message"`
	Fields    json.RawMessage `json:"fields" db:"fields"`
	TimeStamp time.Time       `json:"timestamp" db:"timestamp"`
}

const (
	testEndpoint = "http://localhost:3000/api/v1/"
	testService  = "node"
)

func TestNew(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	tests := []struct {
		name    string
		opts    []AppOption
		want    *App
		wantErr bool
	}{
		{
			name: "custom buffer",
			opts: []AppOption{WithBuffer(500), WithStoreService(testService), WithLogscribeEndpoint(testEndpoint)},
			want: &App{
				StoreInterval:     DefaultLogStoreInterval,
				buffer:            make(chan map[string]any, 500),
				LogscribeEndpoint: testEndpoint,
				Service:           testService,
			},
			wantErr: false,
		},
		{
			name: "custom interval",
			opts: []AppOption{WithStoreInterval(time.Second), WithStoreService(testService), WithLogscribeEndpoint(testEndpoint)},
			want: &App{
				StoreInterval:     time.Second,
				buffer:            make(chan map[string]any, DefaultBufferSize),
				LogscribeEndpoint: testEndpoint,
				Service:           testService,
			},
			wantErr: false,
		},
		{
			name: "negative buffer",
			opts: []AppOption{WithBuffer(-1), WithStoreService(testService), WithLogscribeEndpoint(testEndpoint)},
			want: &App{
				StoreInterval:     DefaultLogStoreInterval,
				buffer:            make(chan map[string]any, DefaultBufferSize), // Should fallback to default
				LogscribeEndpoint: testEndpoint,
				Service:           testService,
			},
			wantErr: false,
		},
		{
			name: "custom logscribe endpoint",
			opts: []AppOption{WithStoreService(testService), WithLogscribeEndpoint(testEndpoint)},
			want: &App{
				StoreInterval:     DefaultLogStoreInterval,
				buffer:            make(chan map[string]any, DefaultBufferSize),
				LogscribeEndpoint: testEndpoint,
				Service:           testService,
			},
			wantErr: false,
		},
		{
			name:    "service not provided",
			opts:    []AppOption{WithLogscribeEndpoint(testEndpoint)},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid logscribe endpoint",
			opts:    []AppOption{WithLogscribeEndpoint("test"), WithStoreService(testService)},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.Equal(t, tt.want.StoreInterval, got.StoreInterval, "StoreInterval should be equal")
				assert.Equal(t, cap(tt.want.buffer), cap(got.buffer), "Buffer capacity should be equal")
			}
		})
	}
}

func TestLogscribeConsumerWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	tests := []struct {
		name    string
		log     []byte
		wantErr bool
	}{
		{
			name: "write log",
			log:  []byte("{\"test\": \"test\", \"level\": \"error\"}"),
		},
		{
			name:    "invalid json",
			log:     []byte("{test: \"test\"}"),
			wantErr: true,
		},
		{
			name:    "empty log",
			log:     []byte("{}"),
			wantErr: true,
		},
		{
			name: "large log",
			log:  []byte("{\"test\": \"" + strings.Repeat("test", 10000) + "\", \"level\": \"error\"}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, _ := New(WithStoreService(testService), WithLogscribeEndpoint(testEndpoint))
			_, err := l.Write(tt.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogscribeConsumer.Write() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				res := <-l.buffer
				assert.JSONEq(t, string(tt.log), string(mapToJSON(res)))
			}
		})
	}
}

func TestBulkCopyLogEntries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	app, _ := New(WithLogscribeEndpoint("http://localhost:3000/api/v1/"), WithStoreService("node"))

	events := []map[string]any{
		{
			"time":    1234567890.0,
			"level":   "error",
			"message": "test message",
			"service": "node",
			"field1":  "test field 1",
			"field2":  123,
		},
		{
			"time":    9876543210.0,
			"level":   "info",
			"message": "another test message",
			"service": "dal",
			"field3":  "test field 3",
			"field4":  456,
		},
		{
			"time":    1111111111.0,
			"level":   "debug",
			"message": "debug message",
			"service": "reporter",
			"field5":  "test field 5",
		},
		{
			"time":    1234567890.0,
			"level":   "error",
			"message": "test message",
			"nested": map[string]interface{}{
				"inner": "value",
			},
		},
		{},
	}

	err := app.bulkPostLogEntries(events)
	assert.NoError(t, err)
}

func TestExtractLogscribeEntry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	tests := []struct {
		name     string
		event    map[string]interface{}
		expected *LogInsertModel
		wantErr  bool
	}{
		{
			name: "valid log entry",
			event: map[string]interface{}{
				"time":    1234567890.0,
				"level":   "error",
				"message": "test message",
				"field1":  "test field 1",
				"field2":  123,
			},
			expected: &LogInsertModel{
				Timestamp: time.Unix(1234567890, 0),
				Level:     int(zerolog.ErrorLevel),
				Message:   "test message",
				Service:   testService,
				Fields:    json.RawMessage(`{"field1":"test field 1","field2":123}`),
			},
		},
		{
			name: "missing fields",
			event: map[string]interface{}{
				"time":    1234567890.0,
				"message": "test message",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "invalid level",
			event: map[string]interface{}{
				"time":    1234567890.0,
				"level":   "unknown",
				"message": "test message",
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "nested fields",
			event: map[string]interface{}{
				"time":    1234567890.0,
				"level":   "error",
				"message": "test message",
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
			expected: &LogInsertModel{
				Timestamp: time.Unix(1234567890, 0),
				Level:     int(zerolog.ErrorLevel),
				Message:   "test message",
				Service:   testService,
				Fields:    json.RawMessage(`{"nested":{"inner":"value"}}`),
			},
		},
	}

	app, err := New(WithStoreService(testService), WithLogscribeEndpoint(testEndpoint))
	if err != nil {
		t.Errorf("failed to create logscribe app: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := app.extractLogscribeEntry(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractDbEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestPostToLogscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	app, err := New(
		WithStoreInterval(100*time.Millisecond),
		WithLogscribeEndpoint("http://localhost:3000/api/v1/"),
		WithStoreService("test"),
	)
	if err != nil {
		t.Errorf("failed to create logscribeconsumer app: %v", err)
	}
	go app.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	log.Error().Msg("this message should be posted to logscribe")
	time.Sleep(2 * BulkLogsCopyInterval)

	res, err := db.QueryRow[Count](ctx, "SELECT Count(*) FROM logs;", nil)
	if err != nil {
		t.Errorf("failed to query logs: %v", err)
	}
	assert.Equal(t, 1, res.Count)
}

func TestNotPostToLogscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	app, err := New(
		WithStoreInterval(100*time.Millisecond),
		WithLogscribeEndpoint("http://localhost:3000/api/v1/"),
		WithStoreService("test"),
		WithPostToLogscribe(false),
	)
	if err != nil {
		t.Errorf("failed to create logscribeconsumer app: %v", err)
	}
	go app.Run(ctx)
	time.Sleep(500 * time.Millisecond)
	log.Error().Msg("this message shouldn't be posted to logscribe")
	time.Sleep(2 * BulkLogsCopyInterval)

	res, err := db.QueryRow[Count](ctx, "SELECT Count(*) FROM logs;", nil)
	if err != nil {
		t.Errorf("failed to query logs: %v", err)
	}
	assert.Equal(t, 0, res.Count)
}

func TestCustomLogLevel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startLogscribe(ctx, t)
	defer cleanup(ctx)

	zerologLevel := zerolog.InfoLevel
	app, err := New(
		WithStoreInterval(100*time.Millisecond),
		WithLogscribeEndpoint("http://localhost:3000/api/v1/"),
		WithStoreService("test"),
		WithPostToLogscribe(false),
		WithStoreLevel(zerologLevel.String()),
	)
	if err != nil {
		t.Errorf("failed to create logscribeconsumer app: %v", err)
	}
	go app.Run(ctx)
	time.Sleep(500 * time.Millisecond)

	log.Debug().Msg("debug message")
	log.Info().Msg("info message")
	log.Warn().Msg("warn message")
	log.Error().Msg("error message")
	time.Sleep(2 * BulkLogsCopyInterval)

	res, err := db.QueryRows[LogInsertModel](ctx, "SELECT * FROM logs;", nil)
	if err != nil {
		t.Errorf("failed to query logs: %v", err)
	}
	for _, log := range res {
		assert.GreaterOrEqual(t, zerologLevel, zerolog.Level(log.Level))
	}
}

func mapToJSON(m map[string]any) string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}
