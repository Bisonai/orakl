package main

import (
	"context"

	"bisonai.com/miko/node/pkg/boot"
	"bisonai.com/miko/node/pkg/utils/loginit"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loginit.InitZeroLog()

	err := boot.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start bootnode")
		return
	}
}
