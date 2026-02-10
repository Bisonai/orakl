package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"bisonai.com/miko/node/pkg/checker/balance"
	"bisonai.com/miko/node/pkg/checker/dal"
	"bisonai.com/miko/node/pkg/checker/dalstats"
	"bisonai.com/miko/node/pkg/checker/dbcronjob"
	"bisonai.com/miko/node/pkg/checker/health"
	"bisonai.com/miko/node/pkg/checker/inspect"
	"bisonai.com/miko/node/pkg/checker/offset"
	"bisonai.com/miko/node/pkg/checker/peers"
	"bisonai.com/miko/node/pkg/checker/price"
	"bisonai.com/miko/node/pkg/checker/signer"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/loginit"
)

func main() {
	ctx := context.Background()

	loginit.InitZeroLog()

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

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := inspect.Start(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error starting inspect checker")
			os.Exit(1)
		}
	}()

	log.Info().Msg("inspect checker started")

	runPriceChecker, _ := strconv.ParseBool(os.Getenv("RUN_PRICE_CHECKER"))
	if runPriceChecker {
		dalApiKey := secrets.GetSecret("DAL_API_KEY")
		slackEndpoint := secrets.GetSecret("PRICE_CHECK_SLACK")
		trackingPairsRaw := os.Getenv("PRICE_CHECK_PAIRS")
		trackingPairs := strings.Split(trackingPairsRaw, ",")
		for i := range trackingPairs {
			trackingPairs[i] = strings.TrimSpace(trackingPairs[i])
		}

		thresholdRaw := os.Getenv("PRICE_DIFF_THRESHOLD")
		threshold, err := strconv.ParseFloat(thresholdRaw, 64)
		if err != nil {
			log.Error().Err(err).Msg("error parsing price diff threshold, using default")
			threshold = 0
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := price.Start(
				ctx,
				price.WithDalApiKey(dalApiKey),
				price.WithSlackEndpoint(slackEndpoint),
				price.WithTrackingPairs(trackingPairs),
				price.WithpriceDiffThreshold(threshold),
			)
			if err != nil {
				log.Error().Err(err).Msg("error starting price checker")
				os.Exit(1)
			}
		}()
	}

	wg.Wait()
}
