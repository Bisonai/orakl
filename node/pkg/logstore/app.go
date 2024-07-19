package logstore

import (
	"context"
	"encoding/json"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(options ...LogStoreOption) *LogStore {
	c := &LogStoreConfig{
		StoreInterval: DefaultLogStoreInterval,
		Buffer:        1000,
	}
	for _, option := range options {
		option(c)
	}
	return &LogStore{
		StoreInterval: c.StoreInterval,
		logChannel:    make(chan []byte, c.Buffer),
		logEntries:    make([][]byte, 0),
	}
}

func (l *LogStore) Write(p []byte) (n int, err error) {
	l.logChannel <- p
	return len(p), nil
}

func (l *LogStore) Run(ctx context.Context) {
	ticker := time.NewTicker(l.StoreInterval)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-l.logChannel:
			if !ok {
				l.processBatch(ctx)
				return
			}
			l.logEntries = append(l.logEntries, entry)
		case <-ticker.C:
			if len(l.logEntries) > 0 {
				l.processBatch(ctx)
			}
		}
	}
}

func (l *LogStore) processBatch(ctx context.Context) {
	defer func() {
		l.logEntries = make([][]byte, 0)
	}()

	if len(l.logEntries) == 0 {
		return
	}

	bulkCopyEntries := [][]any{}
	for _, entry := range l.logEntries {
		var event map[string]interface{}
		if err := json.Unmarshal(entry, &event); err != nil {
			log.Error().Err(err).Msg("Error unmarshaling log entry")
			continue
		}

		res, err := extractLogEntry(event)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting log entry")
			continue
		}

		if res == nil {
			continue
		}

		bulkCopyEntries = append(bulkCopyEntries, res)
	}
	if len(bulkCopyEntries) > 0 {
		_, err := db.BulkCopy(ctx, "zerologs", []string{"timestamp", "level", "message", "fields"}, bulkCopyEntries)
		if err != nil {
			log.Error().Err(err).Msg("Error bulk copying log entries")
		}
	}
}

func extractLogEntry(event map[string]interface{}) ([]any, error) {
	timestamp := time.Unix(int64(event["time"].(float64)), 0)
	message := event["message"].(string)
	levelStr := event["level"].(string)
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	if level < zerolog.ErrorLevel {
		return nil, nil
	}

	delete(event, "time")
	delete(event, "level")
	delete(event, "message")

	jsonData, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	fields := json.RawMessage(jsonData)
	return []any{timestamp, level, message, fields}, nil
}
