package main

import (
	"context"
	"errors"
	"time"

	"bisonai.com/orakl/node/pkg/libp2p"
	"bisonai.com/orakl/node/pkg/raft"
	"github.com/rs/zerolog/log"
)

// its purpose is to quickly run raft node without specific functionality
// should be enough to test leader election, resign, relaction and so on.

func main() {
	ctx := context.Background()

	host1, err := libp2p.MakeHost(10001)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to make host")
	}
	host2, err := libp2p.MakeHost(10002)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to make host")
	}
	host1.Connect(ctx, host2.Peerstore().PeerInfo(host2.ID()))
	host2.Connect(ctx, host1.Peerstore().PeerInfo(host1.ID()))

	ps1, err := libp2p.MakePubsub(ctx, host1)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to make pubsub")
	}
	ps2, err := libp2p.MakePubsub(ctx, host2)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to make pubsub")
	}

	topicString := "orakl-test-raft"
	topic1, err := ps1.Join(topicString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to join topic")
	}

	topic2, err := ps2.Join(topicString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to join topic")
	}

	node1 := raft.NewRaftNode(host1, ps1, topic1, 100, 3*time.Second)
	node1.LeaderJob = func() error {
		log.Debug().Int("Term", node1.GetCurrentTerm()).Msg("node 1 Leader job")
		node1.IncreaseTerm()
		return nil
	}
	node1.HandleCustomMessage = func(message raft.Message) error {
		log.Debug().Msg("Custom message")
		return errors.New("unknown message type")
	}

	node2 := raft.NewRaftNode(host2, ps2, topic2, 100, 3*time.Second)
	node2.LeaderJob = func() error {
		log.Debug().Int("Term", node2.GetCurrentTerm()).Msg("node 2 Leader job")
		node2.IncreaseTerm()
		return nil
	}
	node2.HandleCustomMessage = func(message raft.Message) error {
		log.Debug().Msg("Custom message")
		return errors.New("unknown message type")
	}

	go node1.Run(ctx)
	time.Sleep(10 * time.Second)
	go node2.Run(ctx)

	time.Sleep(10 * time.Second)
	go node1.StopHeartbeatTicker()
	time.Sleep(10 * time.Second)
	go node2.StopHeartbeatTicker()
	time.Sleep(10 * time.Second)
}
