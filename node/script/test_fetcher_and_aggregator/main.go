package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/admin"
	"bisonai.com/miko/node/pkg/aggregator"
	"bisonai.com/miko/node/pkg/bus"
	"bisonai.com/miko/node/pkg/fetcher"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/rs/zerolog/log"
)

// its purpose is to check whether api + fetcher + aggregator works properly
// it syncs adapters and aggregators from orakl config
// setting up entries in proxies table before running this script is recommended

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

	time.Sleep(1 * time.Second)

	_, err := http.Post("http://localhost:"+os.Getenv("APP_PORT")+"/api/v1/adapter/sync", "application/json", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sync from orakl config")
		return
	}

	_, err = http.Post("http://localhost:"+os.Getenv("APP_PORT")+"/api/v1/aggregator/sync/adapter", "application/json", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sync from adapter table")
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		bootnode := os.Getenv("BOOT_NODE")
		if bootnode == "" {
			log.Debug().Msg("No bootnode specified")
		}
		listenPort, err := strconv.Atoi(os.Getenv("LISTEN_PORT"))
		if err != nil {
			log.Error().Err(err).Msg("Error parsing LISTEN_PORT")
			return
		}

		host, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch(), libp2pSetup.WithPort(listenPort))
		if err != nil {
			log.Error().Err(err).Msg("Failed to make host")
			return
		}

		ps, err := libp2pSetup.MakePubsub(ctx, host)
		if err != nil {
			log.Error().Err(err).Msg("Failed to make pubsub")
			return
		}

		err = libp2pSetup.ConnectThroughBootApi(ctx, host)
		if err != nil {
			log.Error().Err(err).Msg("Failed to connect through boot api")
			return
		}

		a := aggregator.New(mb, host, ps)
		err = a.Run(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start aggregator")
			return
		}

	}()

	wg.Wait()
}
