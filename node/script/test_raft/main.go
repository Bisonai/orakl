package main

import (
	"context"
	"errors"
	"flag"
	"time"

	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	"bisonai.com/orakl/node/pkg/raft"
	"github.com/rs/zerolog/log"
)

// its purpose is to quickly run raft node without specific functionality
// should be enough to test leader election, resign, relaction and so on.

func main() {
	ctx := context.Background()

	port := flag.Int("p", 0, "libp2p port")
	bootnode := flag.String("b", "", "bootnode")

	flag.Parse()
	if *port == 0 {
		log.Fatal().Msg("Please provide a port to bind on with -p")
	}

	host, ps, err := libp2p_setup.Setup(ctx, *bootnode, *port)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup libp2p")
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
