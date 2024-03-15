package main

import (
	"bisonai.com/orakl/node/pkg/libp2p"
	"bisonai.com/orakl/node/pkg/utils"
	"github.com/rs/zerolog/log"
)

func main() {
	seed, err := utils.EncryptText("orakl-bootnode-test")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to encrypt seed")
	}

	bootnode, err := libp2p.SetBootNode(10010, seed)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to set bootnode")
	}

	if bootnode == nil {
		log.Fatal().Msg("Bootnode is nil")
	}

	// Block indefinitely
	select {}
}
