package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"

	"bisonai.com/orakl/node/pkg/checker/balance"
	"bisonai.com/orakl/node/pkg/checker/dal"
	"bisonai.com/orakl/node/pkg/checker/dalstats"
	"bisonai.com/orakl/node/pkg/checker/dbcronjob"
	"bisonai.com/orakl/node/pkg/checker/event"
	"bisonai.com/orakl/node/pkg/checker/health"
	"bisonai.com/orakl/node/pkg/checker/offset"
	"bisonai.com/orakl/node/pkg/checker/peers"
	"bisonai.com/orakl/node/pkg/checker/signer"
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
		port := os.Getenv("POR_PORT")
		if port == "" {
			port = "3000"
		}

		http.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
			// Respond with a simple string
			_, err := w.Write([]byte("Orakl Sentinel"))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
		})

		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal().Err(err).Msg("failed to start http server")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := balance.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting balance checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("balance checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := health.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting health checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("health checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := event.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting event checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("event checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := signer.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting signer checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("signer checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := peers.Start()
		if err != nil {
			log.Error().Err(err).Msg("error starting peers checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("peers checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dal.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting dal checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("dal checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dalstats.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting dalstats checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("dal stats checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dbcronjob.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting dbcronjob checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("dbcronjob checker started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := offset.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting offset checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("offset checker started")

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
