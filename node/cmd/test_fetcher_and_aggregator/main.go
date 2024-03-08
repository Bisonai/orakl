package main

import (
	"context"
	"os"
	"strconv"
	"sync"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/fetcher"
	"bisonai.com/orakl/node/pkg/libp2p"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	mb := bus.New(10)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := admin.Run(mb)
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		bootnode := os.Getenv("BOOT_NODE")
		if bootnode == "" {
			log.Fatal().Msg("No bootnode specified")
			return
		}
		listenPort, err := strconv.Atoi(os.Getenv("LISTEN_PORT"))
		if err != nil {
			log.Error().Err(err).Msg("Error parsing LISTEN_PORT")
			return
		}

		host, ps, err := libp2p.Setup(ctx, bootnode, listenPort)
		if err != nil {
			log.Error().Err(err).Msg("Failed to setup libp2p")
			return
		}
		a := aggregator.New(mb, *host, ps)
		err = a.Run(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start aggregator")
			return
		}

	}()

	wg.Wait()
}
