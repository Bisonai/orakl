package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/libp2p/setup"

	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	seed := "orakl-bootnode-test"

	bootnode, err := setup.SetBootNode(ctx, 10010, seed)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set bootnode")
	}

	if bootnode == nil {
		log.Fatal().Msg("Bootnode is nil")
	}

	// Block indefinitely
	select {}
}
