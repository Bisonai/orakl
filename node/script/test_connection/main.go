package main

import (
	"context"
	"flag"
	"time"

	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"github.com/rs/zerolog/log"
)

// run this script in each different vm and check the output for connection time

func main() {
	ctx := context.Background()
	topicString := "orakl-test-discover-connection-time"

	port := flag.Int("p", 10010, "libp2p port")
	flag.Parse()
	if *port == 0 {
		log.Fatal().Msg("Please provide a port to bind on with -p")
	}

	startTime := time.Now()
	_, ps, err := libp2p_setup.SetupFromBootApi(ctx, *port)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup libp2p")
	}

	topic, err := ps.Join(topicString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to join topic")
	}

	_, err = topic.Subscribe()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to subscribe")
	}

	for {
		if len(ps.ListPeers(topicString)) > 0 {
			log.Debug().Str("connection time", time.Since(startTime).String()).Msg("Connected to peers")
			break
		}
	}

	// Block indefinitely
	select {}
}
