package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"bisonai.com/orakl/node/pkg/reporter"
	"bisonai.com/orakl/node/pkg/zeropglog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	zeropglog := zeropglog.New()
	go zeropglog.Run(ctx)

	r := reporter.New()
	err := r.Run(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start reporter")
		cancel()
		return
	}

	<-sigChan
	log.Info().Msg("Reporter termination signal received")

	r.WsHelper.Close()
	cancel()

	log.Info().Msg("Reporter service has stopped")
}
