//nolint:all
package lograkl

import (
	"context"
	"encoding/json"
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
				logChannel:    make(chan map[string]any, 1000),
				logEntries:    []map[string]any{},
			},
		},
		{
			name: "custom buffer",
			opts: []AppOption{WithBuffer(500)},
			want: &App{
				StoreInterval: DefaultLogStoreInterval,
				logChannel:    make(chan map[string]any, 500),
				logEntries:    []map[string]any{},
			},
		},
		{
			name: "custom interval",
			opts: []AppOption{WithStoreInterval(time.Second)},
			want: &App{
				StoreInterval: time.Second,
				logChannel:    make(chan map[string]any, 1000),
				logEntries:    []map[string]any{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.opts...)
			assert.ObjectsAreEqual(tt.want, got)
		})
	}
}

func TestLogStoreWrite(t *testing.T) {
	tests := []struct {
		name    string
		log     []byte
		wantErr bool
	}{
		{
			name: "write log",
			log:  []byte("{\"test\": \"test\"}"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()
			_, err := l.Write(tt.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("LogStore.Write() error = %v, wantErr %v", err, tt.wantErr)
			}
			res := <-l.logChannel

			assert.Equal(t, map[string]any{"test": "test"}, res)
		})
	}
}

func TestLograklRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lograkl := New()
	go lograkl.Run(ctx)
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, 0, len(lograkl.logEntries))

	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lograkl = New()
	lograkl.logChannel <- map[string]any{"test": "test"}
	go lograkl.Run(ctx)
	time.Sleep(time.Millisecond * 200)
	assert.Equal(t, 1, len(lograkl.logEntries))
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
	}

	expected := []LogEntry{
		{
			Level:     zerolog.ErrorLevel,
			Message:   "test message",
			Fields:    json.RawMessage(`{"field1": "test field 1", "field2": 123}`),
			TimeStamp: time.Unix(1234567890, 0),
		},
	}

	lograkl := New()
	lograkl.logEntries = []map[string]any{events[0], events[1]}

	err := lograkl.bulkCopyLogEntries(ctx)
	assert.NoError(t, err)

	result, err := db.QueryRows[LogEntry](ctx, "SELECT level, message, fields, timestamp FROM zerologs", nil)
	assert.NoError(t, err)

	assert.Equal(t, expected, result)

	err = db.QueryWithoutResult(ctx, "DELETE FROM zerologs", nil)
	assert.NoError(t, err)

}

func TestExtractLogEntry(t *testing.T) {
	event := map[string]interface{}{
		"time":    1234567890.0,
		"level":   "error",
		"message": "test message",
		"field1":  "test field 1",
		"field2":  123,
	}

	expected := []any{
		time.Unix(1234567890, 0),
		zerolog.ErrorLevel,
		"test message",
		json.RawMessage(`{"field1":"test field 1","field2":123}`),
	}

	actual, err := extractDbEntry(event)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
