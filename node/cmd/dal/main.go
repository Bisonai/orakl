package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/dal"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := dal.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start DAL")
	}
}
