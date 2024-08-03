package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/bus"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	mb := bus.New(10)
	err := admin.Run(ctx, mb)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start admin server")
		return
	}
}
