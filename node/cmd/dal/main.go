package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/dal"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	err := dal.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start dalapi")
		return
	}
}
