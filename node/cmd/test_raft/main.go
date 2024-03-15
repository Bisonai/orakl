package main

import (
	"context"
	"errors"
	"flag"
	"time"

	"bisonai.com/orakl/node/pkg/libp2p"
	"bisonai.com/orakl/node/pkg/raft"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	port := flag.Int("p", 0, "libp2p port")
	flag.Parse()
	if *port == 0 {
		log.Fatal().Msg("Please provide a port to bind on with -p")
	}

	host, ps, err := libp2p.Setup(ctx, "", *port)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup libp2p")
		return
	}

	topicString := "orakl-raft-test-topic"
	topic, err := ps.Join(topicString)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to join topic")
	}

	node := raft.NewRaftNode(*host, ps, topic, 100, 5*time.Second)
	node.LeaderJob = func() error {
		log.Debug().Int("Term", node.GetCurrentTerm()).Msg("Leader job")
		node.IncreaseTerm()
		return nil
	}
	node.HandleCustomMessage = func(message raft.Message) error {
		log.Debug().Msg("Custom message")
		return errors.New("unknown message type")
	}

	node.Run(ctx)
}
