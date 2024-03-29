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

// it assumes that boot node is running in `BOOT_API_URL` or `http://localhost:8089`

func main() {
	ctx := context.Background()

	port := flag.Int("p", 10010, "libp2p port")

	flag.Parse()

	host, ps, err := libp2p_setup.SetupFromBootApi(ctx, *port)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup libp2p")
	}

	log.Debug().Msg("connecting to topic string")
	topicString := "orakl-raft-test-topic"
	topic, err := ps.Join(topicString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to join topic")
	}
	log.Debug().Msg("connected to topic string")

	log.Debug().Msg("creating raft node")
	node := raft.NewRaftNode(host, ps, topic, 100, 5*time.Second)
	node.LeaderJob = func() error {
		log.Debug().Int("subscribers", node.SubscribersCount()).Int("Term", node.GetCurrentTerm()).Msg("Leader job")
		node.IncreaseTerm()
		return nil
	}

	node.HandleCustomMessage = func(message raft.Message) error {
		log.Debug().Msg("Custom message")
		return errors.New("unknown message type")
	}

	log.Debug().Msg("running raft node")
	node.Run(ctx)
}
