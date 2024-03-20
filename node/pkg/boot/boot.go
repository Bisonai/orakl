package boot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"bisonai.com/orakl/node/pkg/boot/peer"
	"bisonai.com/orakl/node/pkg/boot/utils"
	"bisonai.com/orakl/node/pkg/db"
	libp2p_setup "bisonai.com/orakl/node/pkg/libp2p/setup"
	libp2p_utils "bisonai.com/orakl/node/pkg/libp2p/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

const REFRESH_INTERVAL = 60 * time.Second

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

	refreshTimer := time.NewTimer(REFRESH_INTERVAL)
	go func() {
		for {
			select {
			case <-refreshTimer.C:
				err = RefreshJob(ctx)
				if err != nil {
					log.Error().Err(err).Msg("Failed to refresh peers")
				}
			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				refreshTimer.Stop()
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
	peers, err := db.QueryRows[peer.PeerModel](ctx, peer.GetPeer, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get peers")
		return err
	}

	if len(peers) == 0 {
		return nil
	}

	bootPortStr := os.Getenv("BOOT_LISTEN_PORT")
	if bootPortStr == "" {
		log.Info().Msg("BOOT_PORT not set, defaulting to 10010")
		bootPortStr = "10010"
	}

	bootPort, err := strconv.Atoi(bootPortStr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to convert BOOT_PORT to int")
		bootPort = 10010
	}

	h, err := libp2p_setup.MakeHost(bootPort)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make host")
		return err
	}

	for _, p := range peers {
		connectionUrl := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", p.Ip, p.Port, p.HostId)
		isAlive, liveCheckErr := libp2p_utils.IsHostAlive(ctx, h, connectionUrl)
		if liveCheckErr != nil {
			log.Error().Err(liveCheckErr).Msg("Failed to check peer")
			if liveCheckErr.Error() != "failed to connect to peer" {
				continue
			}
		}
		if isAlive {
			continue
		}

		log.Info().Str("peer", connectionUrl).Msg("Peer is not alive")
		err = db.QueryWithoutResult(ctx, peer.DeletePeerById, map[string]any{"id": p.Id})
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
