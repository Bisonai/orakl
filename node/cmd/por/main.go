package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"bisonai.com/orakl/node/pkg/por"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	err := logscribeconsumer.Start(ctx, "por")
	if err != nil {
		log.Error().Err(err).Msg("Failed to start logscribe consumer")
		return
	}

	app, err := por.New(ctx)
	if err != nil {
		panic(err)
	}
	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}
}
