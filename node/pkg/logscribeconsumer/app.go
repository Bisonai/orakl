package logscribeconsumer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(options ...AppOption) (*App, error) {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}

	c := &AppConfig{
		StoreInterval:     DefaultLogStoreInterval,
		Buffer:            DefaultBufferSize,
		Level:             os.Getenv("LOGSCRIBE_LOG_LEVEL"),
		PostToLogscribe:   true,
		LogscribeEndpoint: DefaultLogscribeEndpoint,
	}
	for _, option := range options {
		option(c)
	}

	resp, err := request.RequestRaw(request.WithEndpoint(c.LogscribeEndpoint))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errorsentinel.ErrLogscribeConsumerEndpointUnresponsive
	}

	if c.Service == "" {
		log.Error().Msg("Service not provided")
		return nil, errorsentinel.ErrLogscribeConsumerServiceNotProvided
	}
	level, err := zerolog.ParseLevel(c.Level)
	if err != nil {
		level = DefaultLogStoreLevel
	}

	return &App{
		StoreInterval:     c.StoreInterval,
		buffer:            make(chan map[string]any, c.Buffer),
		consoleWriter:     consoleWriter,
		LogscribeEndpoint: c.LogscribeEndpoint,
		Service:           c.Service,
		Level:             level,
		PostToLogscribe:   c.PostToLogscribe,
	}, nil
}

func (a *App) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	_, err = a.consoleWriter.Write(p)
	if err != nil {
		return 0, err
	}

	if !a.PostToLogscribe {
		return len(p), nil
	}

	res, err := byte2Entry(p)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, errorsentinel.ErrLogEmptyLogByte
	}

	if !isLogLevelValid(res, a.Level) {
		return 0, nil
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
			errors := a.processBatch(ctx)
			for _, err := range errors {
				log.Err(err).Msg("log batch process failure")
			}

		}
	}
}

func (a *App) setup() {
	consoleLogLevel, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		consoleLogLevel = DefaultLogConsoleLevel
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(consoleLogLevel)
	logger := zerolog.New(a).With().Timestamp().Logger()
	log.Logger = logger
}

func (a *App) processBatch(ctx context.Context) []error {
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
		batchLen := len(batch)
		errors := []error{}
		for i := 0; i < batchLen; i += ProcessLogsBatchSize {
			end := i + ProcessLogsBatchSize
			if end > batchLen {
				end = batchLen
			}
			err := a.bulkPostLogEntries(batch[i:end])
			if err != nil {
				errors = append(errors, err)
			}
		}
		return errors
	default:
		return nil
	}
}

func isLogLevelValid(entry map[string]any, minLevel zerolog.Level) bool {
	levelStr, ok := entry["level"].(string)
	if !ok {
		return false
	}

	lv, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return false
	}

	return lv >= minLevel
}

func (a *App) bulkPostLogEntries(logEntries []map[string]any) error {
	bulkPostEntries := []LogInsertModel{}

	for _, entry := range logEntries {
		res, err := a.extractLogscribeEntry(entry)
		if err != nil || res == nil {
			log.Error().Err(err).Msg("Error extracting log entry")
			continue
		}

		bulkPostEntries = append(bulkPostEntries, *res)
	}

	if len(bulkPostEntries) > 0 {
		res, err := request.RequestRaw(request.WithEndpoint(a.LogscribeEndpoint), request.WithBody(bulkPostEntries), request.WithMethod("POST"))
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

func (a *App) extractLogscribeEntry(entry map[string]interface{}) (*LogInsertModel, error) {
	timeVal, ok := entry["time"]
	if !ok {
		return nil, errorsentinel.ErrLogTimestampNotExist
	}
	messageVal, ok := entry["message"]
	if !ok {
		return nil, errorsentinel.ErrLogMsgNotExist
	}
	levelStr, ok := entry["level"].(string)
	if !ok {
		return nil, errorsentinel.ErrLogLvlNotExist
	}

	timestamp := time.Unix(int64(timeVal.(float64)), 0)
	message := messageVal.(string)

	zerologLevel, err := zerolog.ParseLevel(levelStr)
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
	return &LogInsertModel{Timestamp: timestamp, Service: a.Service, Level: int(zerologLevel), Message: message, Fields: fields}, nil
}

func byte2Entry(b []byte) (map[string]any, error) {
	var entry map[string]interface{}
	if err := json.Unmarshal(b, &entry); err != nil {
		log.Error().Err(err).Bytes("raw_entry", b).Msg("Error unmarshaling log entry")
		return nil, err
	}
	return entry, nil
}
