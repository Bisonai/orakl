package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"

	"bisonai.com/orakl/node/pkg/checker/balance"
	"bisonai.com/orakl/node/pkg/checker/dal"
	"bisonai.com/orakl/node/pkg/checker/dalstats"
	"bisonai.com/orakl/node/pkg/checker/dbcronjob"
	"bisonai.com/orakl/node/pkg/checker/event"
	"bisonai.com/orakl/node/pkg/checker/health"
	"bisonai.com/orakl/node/pkg/checker/peers"
	"bisonai.com/orakl/node/pkg/checker/signer"
	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	postToLogscribe, err := strconv.ParseBool(os.Getenv("POST_TO_LOGSCRIBE"))
	if err != nil {
		postToLogscribe = true
	}
	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("sentinel"),
		logscribeconsumer.WithPostToLogscribe(postToLogscribe),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

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

	wg.Wait()
}
