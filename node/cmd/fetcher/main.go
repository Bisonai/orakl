package main

import (
	"context"
	"sync"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/fetcher"
	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("fetcher"),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

	mb := bus.New(10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		adminErr := admin.Run(ctx, mb)
		if adminErr != nil {
			log.Error().Err(adminErr).Msg("Failed to start admin server")
			return
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
			return
		}
	}()
	log.Info().Msg("Fetcher started")
	wg.Wait()
}
