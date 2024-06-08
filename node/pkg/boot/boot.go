package boot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/boot/peer"
	"bisonai.com/orakl/node/pkg/boot/utils"
	"bisonai.com/orakl/node/pkg/db"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	libp2pSetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	libp2pUtils "bisonai.com/orakl/node/pkg/libp2p/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

const REFRESH_INTERVAL = 10 * time.Second

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting boot server")
	app, err := utils.Setup(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup boot server")
		return err
	}

	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Node Boot API")
	})

	peer.Routes(v1)

	port := os.Getenv("BOOT_API_PORT")
	if port == "" {
		port = "8089"
	}

	refreshTicker := time.NewTicker(REFRESH_INTERVAL)
	go func() {
		for {
			select {
			case <-refreshTicker.C:
				err = RefreshJob(ctx)
				if err != nil {
					log.Error().Err(err).Msg("Failed to refresh peers")
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
		log.Fatal().Err(err).Msg("Failed to start boot server")
		return err
	}

	err = app.Shutdown()
	if err != nil {
		log.Error().Err(err).Msg("Failed to shutdown boot server")
		return err
	}
	return nil

}

func RefreshJob(ctx context.Context) error {
	log.Info().Msg("Refreshing peers")
	peers, err := db.QueryRows[peer.PeerModel](ctx, peer.GetPeer, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get peers")
		return err
	}

	if len(peers) == 0 {
		return nil
	}

	h, err := libp2pSetup.NewHost(ctx, libp2pSetup.WithHolePunch(), libp2pSetup.WithQuic())
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
		return err
	}

	for _, p := range peers {

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
		err = db.QueryWithoutResult(ctx, peer.DeletePeerById, map[string]any{"id": p.ID})
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete peer")
		}
	}

	err = h.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close host")
		return err
	}

	return nil
}
