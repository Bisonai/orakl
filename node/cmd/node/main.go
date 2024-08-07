package main

import (
	"context"
	"os"
	"strconv"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/fetcher"
	"bisonai.com/orakl/node/pkg/libp2p/helper"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	postToLogscribe, err := strconv.ParseBool(os.Getenv("POST_TO_LOGSCRIBE"))
	if err != nil {
		postToLogscribe = true
	}
	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("node"),
		logscribeconsumer.WithPostToLogscribe(postToLogscribe),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

	mb := bus.New(1000)
	var wg sync.WaitGroup

	host, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch())
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
	}

	ps, err := libp2pSetup.MakePubsub(ctx, host)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make pubsub")
	}

	err = retrier.Retry(func() error {
		return libp2pSetup.ConnectThroughBootApi(ctx, host)
	}, 5, 10*time.Second, 30*time.Second)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup libp2p")
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		adminErr := admin.Run(ctx, mb)
		if adminErr != nil {
			log.Error().Err(adminErr).Msg("Failed to start admin server")
			os.Exit(1)
		}
	}()

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
			os.Exit(1)
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
			os.Exit(1)
		}
	}()
	log.Info().Msg("Aggregator started")

	wg.Add(1)
	go func() {
		defer wg.Done()
		helperApp := helper.New(mb, host)
		libp2pHelperErr := helperApp.Run(ctx)
		if libp2pHelperErr != nil {
			log.Error().Err(libp2pHelperErr).Msg("Failed to start libp2p helper")
			os.Exit(1)
		}
	}()
	log.Info().Msg("libp2p helper started")

	wg.Wait()
}
