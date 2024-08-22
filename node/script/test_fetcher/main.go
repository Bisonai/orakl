package main

import (
	"context"
	"sync"

	"bisonai.com/miko/node/pkg/admin"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/fetcher"
	"github.com/rs/zerolog/log"
)

// its purpose is to check whether api + fetcher works properly
// it doesn't automatically import adapters so please manually add before running the script

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
