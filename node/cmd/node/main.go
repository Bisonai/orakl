package main

import (
	"context"
	"os"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/admin"
	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/checker/ping"
	"bisonai.com/miko/node/pkg/fetcher"
	"bisonai.com/miko/node/pkg/libp2p/helper"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"bisonai.com/miko/node/pkg/utils/loginit"
	"bisonai.com/miko/node/pkg/utils/retrier"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	loginit.InitZeroLog()

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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Any("panic", r).Msg("panic recovered from network checks")
			}
		}()
		time.Sleep(10 * time.Second) // give some buffer until the app is ready
		ping.Run(ctx)
		os.Exit(1)
	}()

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
	err = admin.SyncMikoConfig(ctx)
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
