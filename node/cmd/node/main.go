package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/fetcher"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
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
	var wg sync.WaitGroup

	bootnode := os.Getenv("BOOT_NODE")
	if bootnode == "" {
		log.Debug().Msg("No bootnode specified")
	}
	listenPort, err := strconv.Atoi(os.Getenv("LISTEN_PORT"))
	if err != nil {
		log.Error().Err(err).Msg("Error parsing LISTEN_PORT")
		return
	}

	host, ps, err := libp2pSetup.SetupFromBootApi(ctx, listenPort)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup libp2p")
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		adminErr := admin.Run(mb)
		if adminErr != nil {
			log.Error().Err(adminErr).Msg("Failed to start admin server")
			return
		}
	}()

	port := os.Getenv("APP_PORT")
	if port == "" {
		log.Info().Msg("No APP_PORT specified, using default 8088")
		port = "8088"
	}

	if err = waitForApi(port); err != nil {
		log.Error().Err(err).Msg("Failed to wait api")
		return
	}
	log.Info().Msg("API is live")

	log.Info().Msg("Syncing orakl config")
	err = admin.SyncOraklConfig(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sync orakl config")
		return
	}
	log.Info().Msg("Orakl config synced")

	wg.Add(1)
	go func() {
		defer wg.Done()
		f := fetcher.New(mb)
		fetcherErr := f.Run(ctx)
		if fetcherErr != nil {
			log.Error().Err(fetcherErr).Msg("Failed to start fetcher")
			return
		}
	}()
	log.Info().Msg("Fetcher started")

	wg.Add(1)
	go func() {
		defer wg.Done()

		a := aggregator.New(mb, host, ps)
		aggregatorErr := a.Run(ctx)
		if aggregatorErr != nil {
			log.Error().Err(aggregatorErr).Msg("Failed to start aggregator")
			return
		}
	}()
	log.Info().Msg("Aggregator started")

	wg.Add(1)
	go func() {
		defer wg.Done()

		r := reporter.New(mb, host, ps)
		reporterErr := r.Run(ctx)
		if reporterErr != nil {
			log.Error().Err(reporterErr).Msg("Failed to start reporter")
			return
		}
	}()
	log.Info().Msg("Reporter started")

	wg.Wait()
}

func waitForApi(port string) error {
	syncUrl := "http://localhost:" + port + "/api/v1"
	for i := 0; i < 10; i++ {
		resp, err := http.Get(syncUrl)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("API did not become live within the expected time")
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
