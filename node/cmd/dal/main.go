package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/dalapi"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	err := dalapi.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start dalapi")
		return
	}
}
