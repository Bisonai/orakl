package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/dal"
	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("dal"),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

	err = dal.Run(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start DAL")
	}
}
