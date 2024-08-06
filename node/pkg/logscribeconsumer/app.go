package logscribeconsumer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(options ...AppOption) *App {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	c := &AppConfig{
		StoreInterval:    DefaultLogStoreInterval,
		Buffer:           DefaultBufferSize,
		LogscribeEnpoint: DefaultLogscribeEnpoint,
	}
	for _, option := range options {
		option(c)
	}
	return &App{
		StoreInterval:    c.StoreInterval,
		buffer:           make(chan map[string]any, c.Buffer),
		consoleWriter:    consoleWriter,
		LogscribeEnpoint: c.LogscribeEnpoint,
	}
}

func (a *App) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	_, err = a.consoleWriter.Write(p)
	if err != nil {
		return 0, err
	}

	res, err := byte2Entry(p)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, errorsentinel.ErrLogEmptyLogByte
	}

	if !isLogLevelValid(res) {
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
	logger := zerolog.New(a).With().Timestamp().Logger()
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
		return a.bulkCopyLogEntries(batch)
	default:
		return nil
	}
}

func isLogLevelValid(entry map[string]any) bool {
	levelStr, ok := entry["level"].(string)
	if !ok {
		return false
	}

	lv, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return false
	}

	if lv < MinimalLogStoreLevel {
		return false
	}
	return true
}

func (a *App) bulkCopyLogEntries(logEntries []map[string]any) error {
	bulkCopyEntries := []LogInsertModel{}

	for _, entry := range logEntries {
		res, err := extractLogscribeEntry(entry)
		if err != nil || res == nil {
			log.Error().Err(err).Msg("Error extracting log entry")
			continue
		}

		bulkCopyEntries = append(bulkCopyEntries, *res)
	}

	if len(bulkCopyEntries) > 0 {
		res, err := request.RequestRaw(request.WithEndpoint(a.LogscribeEnpoint), request.WithBody(bulkCopyEntries), request.WithMethod("POST"))
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return errorsentinel.ErrLogscribeInsertFailed
		}
		log.Debug().Msg("Log entries inserted successfully")
	}
	return nil
}

func extractLogscribeEntry(entry map[string]interface{}) (*LogInsertModel, error) {
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
	service := entry["service"].(string)

	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	delete(entry, "time")
	delete(entry, "level")
	delete(entry, "message")
	delete(entry, "service")

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	fields := json.RawMessage(jsonData)
	return &LogInsertModel{Timestamp: timestamp, Service: service, Level: int(level), Message: message, Fields: fields}, nil
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
