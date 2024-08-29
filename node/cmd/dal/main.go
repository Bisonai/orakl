package main

import (
	"context"

	"bisonai.com/miko/node/pkg/dal"
	"bisonai.com/miko/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := logscribeconsumer.Start(ctx, logscribeconsumer.WithStoreService("dal"))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start logscribe consumer")
		return
	}

	err = dal.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start DAL")
	}
}
