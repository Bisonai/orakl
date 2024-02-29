package main

import (
	"context"
	"sync"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/fetcher"
	"github.com/rs/zerolog/log"
)

func main() {
	mb := bus.New(10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := admin.Run("8088", mb)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start admin server")
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		f := fetcher.New(mb)
		err := f.Run(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("Failed to start fetcher")
			return
		}
	}()

	wg.Wait()

}
