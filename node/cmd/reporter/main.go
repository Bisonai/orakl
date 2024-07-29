package main

import (
	"context"
	"net/http"
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

	go func() {
		port := os.Getenv("REPORTER_PORT")
		if port == "" {
			port = "3000"
		}

		http.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("Orakl Reporter"))
			if err != nil {
				log.Error().Err(err).Msg("failed to write response")
			}
		})

		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal().Err(err).Msg("failed to start http server")
		}
	}()

	<-sigChan
	log.Info().Msg("Reporter termination signal received")

	cancel()

	log.Info().Msg("Reporter service has stopped")
}
