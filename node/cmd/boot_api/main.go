package main

import (
	"context"
	"os"
	"strconv"

	"bisonai.com/orakl/node/pkg/boot"
	"bisonai.com/orakl/node/pkg/logscribeconsumer"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	postToLogscribe, err := strconv.ParseBool(os.Getenv("POST_TO_LOGSCRIBE"))
	if err != nil {
		postToLogscribe = true
	}
	logscribeconsumer, err := logscribeconsumer.New(
		logscribeconsumer.WithStoreService("boot_api"),
		logscribeconsumer.WithPostToLogscribe(postToLogscribe),
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
