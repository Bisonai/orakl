package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/libp2p"
	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	seed, err := utils.EncryptText("orakl-bootnode-test")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to encrypt seed")
	}

	bootnode, _, err := libp2p.SetBootNode(ctx, 10010, seed)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set bootnode")
	}

	if bootnode == nil {
		log.Fatal().Msg("Bootnode is nil")
	}

	// Block indefinitely
	select {}
}
