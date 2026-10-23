package boot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/boot/peer"
	"bisonai.com/miko/node/pkg/boot/utils"
	"bisonai.com/miko/node/pkg/boot/utils/peerstore"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	libp2pSetup "bisonai.com/miko/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/miko/node/pkg/libp2p/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

const REFRESH_INTERVAL = 10 * time.Second

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting boot server")

	h, err := libp2pSetup.NewHost(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
		return err
	}

	// a missing file is created empty: nodes re-register through /peer/sync
	store, err := peerstore.OpenFromEnv(ctx, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open peer store")
		return err
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	app, err := utils.Setup(ctx, &h, store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup boot server")
		return err
	}

	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Miko Node Boot API")
	})
	peer.Routes(v1, store)

	port := os.Getenv("BOOT_API_PORT")
	if port == "" {
		port = "8089"
	}

	refreshTicker := time.NewTicker(REFRESH_INTERVAL)
	refreshDone := make(chan struct{})
	// on return: stop the refresh loop and wait for it, before store.Close (defers run LIFO)
	defer func() {
		cancel()
		<-refreshDone
	}()
	go func() {
		defer close(refreshDone)
		for {
			select {
			case <-refreshTicker.C:
				refreshErr := RefreshJob(ctx, h, store)
				if refreshErr != nil {
					log.Error().Err(refreshErr).Msg("Failed to refresh peers")
				}
			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				refreshTicker.Stop()
				return
			}
		}
	}()

	err = app.Listen(fmt.Sprintf(":%s", port))
	if err != nil {
		log.Error().Err(err).Msg("Failed to start boot server")
		return err
	}

	err = app.Shutdown()
	if err != nil {
		log.Error().Err(err).Msg("Failed to shutdown boot server")
		return err
	}
	return nil

}

func RefreshJob(ctx context.Context, h host.Host, store *peerstore.Store) error {
	log.Info().Msg("Refreshing peers")
	peers, err := store.List(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get peers")
		return err
	}

	if len(peers) == 0 {
		return nil
	}

	for _, p := range peers {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		isAlive, liveCheckErr := libp2pUtils.IsHostAlive(ctx, h, p.Url)
		if liveCheckErr != nil {
			log.Error().Err(liveCheckErr).Msg("Failed to check peer")
			if !errors.Is(liveCheckErr, errorSentinel.ErrLibP2pFailToConnectPeer) {
				continue
			}
		}
		if isAlive {
			continue
		}

		log.Info().Str("peer", p.Url).Msg("Peer is not alive")
		_, err = store.DeleteByID(ctx, p.ID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete peer")
		}
	}

	return nil
}
