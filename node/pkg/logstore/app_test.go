//nolint:all
package logstore

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
		opts []LogStoreOption
		want *LogStore
	}{
		{
			name: "default",
			opts: nil,
			want: &LogStore{
				StoreInterval: DefaultLogStoreInterval,
				logChannel:    make(chan []byte, 1000),
				logEntries:    make([][]byte, 0),
			},
		},
		{
			name: "custom buffer",
			opts: []LogStoreOption{WithBuffer(500)},
			want: &LogStore{
				StoreInterval: DefaultLogStoreInterval,
				logChannel:    make(chan []byte, 500),
				logEntries:    make([][]byte, 0),
			},
		},
		{
			name: "custom interval",
			opts: []LogStoreOption{WithStoreInterval(time.Second)},
			want: &LogStore{
				StoreInterval: time.Second,
				logChannel:    make(chan []byte, 1000),
				logEntries:    make([][]byte, 0),
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
			log:  []byte("test"),
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

			assert.Equal(t, tt.log, res)
		})
	}
}

func TestLogStore_Run(t *testing.T) {
	// Test case 1: test with empty log channel
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	logStore := New()
	go logStore.Run(ctx)
	time.Sleep(time.Millisecond * 200) // wait for the logStore to finish running
	assert.Equal(t, 0, len(logStore.logEntries))

	// Test case 2: test with log entries in log channel
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	logStore = New()
	logStore.logChannel <- []byte(`{"test": "test"}`)
	go logStore.Run(ctx)
	time.Sleep(time.Millisecond * 200) // wait for the logStore to finish running
	assert.Equal(t, 1, len(logStore.logEntries))
}

func TestBulkCopyLogEntries(t *testing.T) {
	ctx := context.Background()

	entries := [][]byte{
		[]byte(`{"time": 1234567890, "level": "error", "message": "test message", "field1": "test field 1", "field2": 123}`),
		[]byte(`{"time": 9876543210, "level": "info", "message": "another test message", "field3": "test field 3", "field4": 456}`),
	}

	expected := []LogEntry{
		{
			Level:     zerolog.ErrorLevel,
			Message:   "test message",
			Fields:    json.RawMessage(`{"field1": "test field 1", "field2": 123}`),
			TimeStamp: time.Unix(1234567890, 0),
		},
	}

	err := bulkCopyLogEntries(ctx, entries)
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

	actual, err := extractLogEntry(event)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
