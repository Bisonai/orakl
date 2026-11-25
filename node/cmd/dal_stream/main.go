package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"bisonai.com/orakl/node/pkg/dalstream"
	"bisonai.com/orakl/node/pkg/utils/buildinfo"
	"bisonai.com/orakl/node/pkg/utils/loginit"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	envFile := flag.String("env", "", "env file")
	flag.Parse()

	loginit.InitZeroLog()
	buildinfo.Register("dal-stream")

	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			log.Fatal().Err(err).Str("file", *envFile).Msg("failed to load env file")
		}
	}

	cfg, err := dalstream.ConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid config")
	}

	// SIGTERM/SIGINT close all streams and stop; a second signal exits at once
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	context.AfterFunc(ctx, stop)

	if err := dalstream.Run(ctx, cfg); err != nil {
		log.Fatal().Err(err).Msg("dal-stream failed")
	}
	log.Info().Msg("dal-stream stopped")
}
