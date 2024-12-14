package main

import (
	"context"
	"flag"

	"bisonai.com/miko/node/pkg/dal"
	"bisonai.com/miko/node/pkg/logscribeconsumer"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func LoadEnvFromFile() {
	envFile := flag.String("env", "", "env file")
	flag.Parse()

	if *envFile != "" {
		log.Info().Msgf("loading env file: %s", *envFile)
		err := godotenv.Load(*envFile)
		if err != nil {
			panic(err)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	LoadEnvFromFile()

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
