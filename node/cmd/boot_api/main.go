package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/boot"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	err := boot.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start bootnode")
		return
	}
}
