package main

import (
	"context"
	"sync"

	"bisonai.com/orakl/node/pkg/admin"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/reporter"
	"bisonai.com/orakl/node/pkg/zeropglog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	zeropglog := zeropglog.New()
	go zeropglog.Run(ctx)

	var wg sync.WaitGroup
	mb := bus.New(10)

	wg.Add(1)
	go func() {
		defer wg.Done()
		adminErr := admin.Run(mb)
		if adminErr != nil {
			log.Error().Err(adminErr).Msg("Failed to start admin server")
			return
		}
	}()
	log.Info().Msg("Admin started")

	err := admin.SyncOraklConfig(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sync orakl config")
		return
	}
	log.Info().Msg("Orakl config synced")

	wg.Add(1)
	go func() {
		defer wg.Done()
		r := reporter.New(mb)
		err := r.Run(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start reporter")
			return
		}
	}()
	log.Info().Msg("Reporter started")

	wg.Wait()
}
