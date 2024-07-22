package main

import (
	"context"
	"os"
	"strings"

	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/reporter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	zerolog.SetGlobalLevel(getLogLevel(logLevel))

	ctx := context.Background()
	mb := bus.New(10)
	r := reporter.New(mb)
	err := r.Run(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start reporter")
		os.Exit(1)
	}
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
