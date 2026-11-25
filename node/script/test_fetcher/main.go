package main

import (
	"context"
	"sync"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/fetcher"
	"github.com/rs/zerolog/log"
)

// its purpose is to check whether api + fetcher works properly
// configs/feeds are synced from orakl-config ($CHAIN) before the fetcher starts

func main() {
	ctx := context.Background()
	mb := bus.New(10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := admin.Run(ctx, mb)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start admin server")
			return
		}
	}()

	if err := admin.SyncOraklConfig(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to sync orakl config")
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		f := fetcher.New(mb)
		err := f.Run(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start fetcher")
			return
		}
	}()

	wg.Wait()

}
