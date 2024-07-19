package lograkl

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(options ...AppOption) *App {
	c := &AppConfig{
		StoreInterval: DefaultLogStoreInterval,
		Buffer:        1000,
	}
	for _, option := range options {
		option(c)
	}
	return &App{
		StoreInterval: c.StoreInterval,
		logChannel:    make(chan map[string]any, c.Buffer),
		logEntries:    []map[string]any{},
	}
}

func (a *App) Write(p []byte) (n int, err error) {
	a.logChannel <- byte2Entry(p)
	return len(p), nil
}

func (a *App) Run(ctx context.Context) {
	a.setup()

	ticker := time.NewTicker(a.StoreInterval)
	defer ticker.Stop()
	for {
		select {
		case entry, ok := <-a.logChannel:
			if !ok {
				a.processBatch(ctx)
				return
			}
			a.logEntries = append(a.logEntries, entry)
		case <-ticker.C:
			if len(a.logEntries) > 0 {
				err := a.processBatch(ctx)
				if err != nil {
					log.Error().Err(err).Msg("Error processing log batch")
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) processBatch(ctx context.Context) error {
	if len(a.logEntries) == 0 {
		return nil
	}

	err := a.bulkCopyLogEntries(ctx)
	if err != nil {
		return err
	}

	a.logEntries = []map[string]any{}
	return nil
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

func byte2Entry(b []byte) map[string]any {
	var entry map[string]interface{}
	if err := json.Unmarshal(b, &entry); err != nil {
		log.Error().Err(err).Bytes("raw_entry", b).Msg("Error unmarshaling log entry")
		return nil
	}
	return entry
}

func (a *App) bulkCopyLogEntries(ctx context.Context) error {
	bulkCopyEntries := [][]any{}

	for _, entry := range a.logEntries {
		res, err := extractDbEntry(entry)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting log entry")
			continue
		}

		if res[1].(zerolog.Level) < zerolog.ErrorLevel {
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
	timestamp := time.Unix(int64(entry["time"].(float64)), 0)
	message := entry["message"].(string)
	levelStr := entry["level"].(string)
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
