//nolint:all
package zeropglog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type LogEntry struct {
	Level     zerolog.Level   `json:"level" db:"level"`
	Message   string          `json:"message" db:"message"`
	Fields    json.RawMessage `json:"fields" db:"fields"`
	TimeStamp time.Time       `json:"timestamp" db:"timestamp"`
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		opts []AppOption
		want *App
	}{
		{
			name: "default",
			opts: nil,
			want: &App{
				StoreInterval: DefaultLogStoreInterval,
				buffer:        make(chan map[string]any, 1000),
			},
		},
		{
			name: "custom buffer",
			opts: []AppOption{WithBuffer(500)},
			want: &App{
				StoreInterval: DefaultLogStoreInterval,
				buffer:        make(chan map[string]any, 500),
			},
		},
		{
			name: "custom interval",
			opts: []AppOption{WithStoreInterval(time.Second)},
			want: &App{
				StoreInterval: time.Second,
				buffer:        make(chan map[string]any, 1000),
			},
		},
		{
			name: "negative buffer",
			opts: []AppOption{WithBuffer(-1)},
			want: &App{
				StoreInterval: DefaultLogStoreInterval,
				buffer:        make(chan map[string]any, 1000), // Should fallback to default
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.opts...)

			assert.Equal(t, tt.want.StoreInterval, got.StoreInterval, "StoreInterval should be equal")

			assert.Equal(t, cap(tt.want.buffer), cap(got.buffer), "Buffer capacity should be equal")
		})
	}
}

func TestZeropglogWrite(t *testing.T) {
	tests := []struct {
		name    string
		log     []byte
		wantErr bool
	}{
		{
			name: "write log",
			log:  []byte("{\"test\": \"test\"}"),
		},
		{
			name:    "invalid json",
			log:     []byte("{test: \"test\"}"),
			wantErr: true,
		},
		{
			name: "empty log",
			log:  []byte("{}"),
		},
		{
			name: "large log",
			log:  []byte("{\"test\": \"" + strings.Repeat("test", 10000) + "\"}"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()
			_, err := l.Write(tt.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("Zeropglog.Write() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				res := <-l.buffer
				assert.JSONEq(t, string(tt.log), string(mapToJSON(res)))
			}
		})
	}
}

func TestBulkCopyLogEntries(t *testing.T) {
	ctx := context.Background()

	events := []map[string]any{
		{
			"time":    1234567890.0,
			"level":   "error",
			"message": "test message",
			"field1":  "test field 1",
			"field2":  123,
		},
		{
			"time":    9876543210.0,
			"level":   "info",
			"message": "another test message",
			"field3":  "test field 3",
			"field4":  456,
		},
		{
			"time":    1111111111.0,
			"level":   "debug",
			"message": "debug message",
			"field5":  "test field 5",
		},
	}

	expected := []LogEntry{
		{
			Level:     zerolog.ErrorLevel,
			Message:   "test message",
			Fields:    json.RawMessage(`{"field1": "test field 1", "field2": 123}`),
			TimeStamp: time.Unix(1234567890, 0),
		},
	}

	err := bulkCopyLogEntries(ctx, []map[string]any{events[0], events[1]})
	assert.NoError(t, err)

	result, err := db.QueryRows[LogEntry](ctx, "SELECT level, message, fields, timestamp FROM zerologs", nil)
	assert.NoError(t, err)

	assert.Equal(t, expected, result)

	err = db.QueryWithoutResult(ctx, "DELETE FROM zerologs", nil)
	assert.NoError(t, err)

}

func TestExtractDBEntry(t *testing.T) {
	tests := []struct {
		name     string
		event    map[string]interface{}
		expected []any
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
			expected: []any{
				time.Unix(1234567890, 0),
				zerolog.ErrorLevel,
				"test message",
				json.RawMessage(`{"field1":"test field 1","field2":123}`),
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
			expected: []any{
				time.Unix(1234567890, 0),
				zerolog.ErrorLevel,
				"test message",
				json.RawMessage(`{"nested":{"inner":"value"}}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := extractDbEntry(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractDbEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func mapToJSON(m map[string]any) string {
	b, _ := json.Marshal(m)
	return string(b)
}
