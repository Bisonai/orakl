package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/boot"
	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("boot_api"),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

	err = boot.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start bootnode")
		return
	}
}
