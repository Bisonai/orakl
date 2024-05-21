package main

import (
	"context"
	"os"
	"strings"
	"sync"

	"bisonai.com/orakl/sentinel/pkg/checker/balance"
	"bisonai.com/orakl/sentinel/pkg/checker/event"
	"bisonai.com/orakl/sentinel/pkg/checker/health"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	zerolog.SetGlobalLevel(getLogLevel(logLevel))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		balance.Start(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		health.Start()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := event.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting event checker")
		}
	}()

	wg.Wait()
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
