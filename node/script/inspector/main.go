package main

import (
	"context"
	"os"

	"bisonai.com/orakl/node/pkg/checker/inspect"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	inspector, err := inspect.Setup(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error setting up inspector")
		os.Exit(1)
	}

	result, err := inspector.Inspect(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error inspecting")
		os.Exit(1)
	}

	log.Info().Str("result", result).Msg("Inspector result")
}
