package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/libp2p"
)

func main() {
	discoverString := "orakl-test-discover-2024"
	port := flag.Int("p", 0, "libp2p port")

	flag.Parse()
	if *port == 0 {
		log.Fatal().Msg("Please provide a port to bind on with -p")
	}

	h, err := libp2p.MakeHost(*port)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create libp2p host")
	}

	ps, err := libp2p.MakePubsub(context.Background(), h)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create pubsub")
	}

	log.Debug().Msg("establishing connection")
	go func() {
		if err = libp2p.DiscoverPeers(context.Background(), h, discoverString, ""); err != nil {
			log.Error().Err(err).Msg("Error from DiscoverPeers")
		}
	}()

	aggregator, err := aggregator.NewAggregator(h, ps, "orakl-aggregator-2024-gazuaa")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create aggregator")
	}
	aggregator.Run(context.Background())
}
