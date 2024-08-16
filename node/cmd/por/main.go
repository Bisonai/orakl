package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"bisonai.com/orakl/node/pkg/por"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("por"),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create a new logscribeconsumer instance")
		return
	}
	go logscribeconsumer.Run(ctx)

	app, err := por.New(ctx)
	if err != nil {
		panic(err)
	}
	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}
}
