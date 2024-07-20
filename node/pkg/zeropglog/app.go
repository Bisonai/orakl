package zeropglog

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(options ...AppOption) *App {
	c := &AppConfig{
		StoreInterval: DefaultLogStoreInterval,
		Buffer:        DefaultBufferSize,
	}
	for _, option := range options {
		option(c)
	}
	return &App{
		StoreInterval: c.StoreInterval,
		buffer:        make(chan map[string]any, c.Buffer),
	}
}

func (a *App) Write(p []byte) (n int, err error) {
	res, err := byte2Entry(p)
	if err != nil {
		return 0, err
	}
	a.buffer <- res
	return len(p), nil
}

func (a *App) Run(ctx context.Context) {
	a.setup()

	ticker := time.NewTicker(a.StoreInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := a.processBatch(ctx)
			if err != nil {
				log.Err(err).Msg("log batch process failure")
			}

		}
	}
}

func (a *App) setup() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	a.setLogLevel()
	a.setLogWriter()
}

func (a *App) setLogLevel() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	zerolog.SetGlobalLevel(getLogLevel(logLevel))
}

func (a *App) setLogWriter() {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	multiWriter := zerolog.MultiLevelWriter(consoleWriter, a)
	logger := zerolog.New(multiWriter).With().Timestamp().Logger()
	log.Logger = logger
}

func (a *App) processBatch(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case entry := <-a.buffer:
		batch := []map[string]any{entry}
	loop:
		for {
			select {
			case entry := <-a.buffer:
				batch = append(batch, entry)
			default:
				break loop
			}
		}
		return bulkCopyLogEntries(ctx, batch)
	default:
		return nil
	}
}

func bulkCopyLogEntries(ctx context.Context, logEntries []map[string]any) error {
	bulkCopyEntries := [][]any{}

	for _, entry := range logEntries {
		res, err := extractDbEntry(entry)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting log entry")
			continue
		}

		if res[1].(zerolog.Level) < zerolog.WarnLevel {
			log.Debug().Msg("Skipping low level log entry")
			continue
		}

		bulkCopyEntries = append(bulkCopyEntries, res)
	}

	if len(bulkCopyEntries) > 0 {
		_, err := db.BulkCopy(ctx, "zerologs", []string{"timestamp", "level", "message", "fields"}, bulkCopyEntries)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractDbEntry(entry map[string]interface{}) ([]any, error) {
	timeVal, ok := entry["time"]
	if !ok {
		return nil, errorsentinel.ErrLogTimestampNotExist
	}
	messageVal, ok := entry["message"]
	if !ok {
		return nil, errorsentinel.ErrLogMsgNotExist
	}
	levelStrVal, ok := entry["level"]
	if !ok {
		return nil, errorsentinel.ErrLogLvlNotExist
	}

	timestamp := time.Unix(int64(timeVal.(float64)), 0)
	message := messageVal.(string)
	levelStr := levelStrVal.(string)

	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	delete(entry, "time")
	delete(entry, "level")
	delete(entry, "message")

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	fields := json.RawMessage(jsonData)
	return []any{timestamp, level, message, fields}, nil
}

func byte2Entry(b []byte) (map[string]any, error) {
	var entry map[string]interface{}
	if err := json.Unmarshal(b, &entry); err != nil {
		log.Error().Err(err).Bytes("raw_entry", b).Msg("Error unmarshaling log entry")
		return nil, err
	}
	return entry, nil
}

func getLogLevel(input string) zerolog.Level {
	switch strings.ToLower(input) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
