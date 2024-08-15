package main

import (
	"context"
	"os"
	"strconv"

	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"bisonai.com/orakl/node/pkg/por"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	postToLogscribe, err := strconv.ParseBool(os.Getenv("POST_TO_LOGSCRIBE"))
	if err != nil {
		postToLogscribe = true
	}
	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("por"),
		logscribeconsumer.WithPostToLogscribe(postToLogscribe),
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
