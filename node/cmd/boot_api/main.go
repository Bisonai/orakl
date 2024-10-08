package main

import (
	"context"

	"bisonai.com/miko/node/pkg/boot"
	"bisonai.com/miko/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := logscribeconsumer.Start(ctx, logscribeconsumer.WithStoreService("boot_api"))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start logscribe consumer")
		return
	}

	err = boot.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start bootnode")
		return
	}
}
