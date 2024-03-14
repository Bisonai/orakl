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
	"bisonai.com/orakl/node/pkg/reporter"
	"github.com/rs/zerolog/log"
)

/*
following script does not include initialization of adapters and aggregators
add 1 adapter and then sync aggregator before running the script
*/

func main() {
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

	host, ps, err := libp2p.Setup(ctx, bootnode, listenPort)
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

	wg.Add(1)
	go func() {
		defer wg.Done()

		a := aggregator.New(mb, *host, ps)
		aaggreegatorErr := a.Run(ctx)
		if aaggreegatorErr != nil {
			log.Error().Err(aaggreegatorErr).Msg("Failed to start aggregator")
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		r := reporter.New(mb, *host, ps)
		reporterErr := r.Run(ctx)
		if reporterErr != nil {
			log.Error().Err(reporterErr).Msg("Failed to start reporter")
			return
		}
	}()

	wg.Wait()
}
