package main

import (
	"bisonai.com/orakl/node/pkg/libp2p"
	"github.com/rs/zerolog/log"
)

func main() {
	bootnode, err := libp2p.SetBootNode(10010)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set bootnode")
	}

	if bootnode == nil {
		log.Fatal().Msg("Bootnode is nil")
	}

	// Block indefinitely
	select {}
}
