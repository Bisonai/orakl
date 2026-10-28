package config

import (
	"context"

	"bisonai.com/miko/node/pkg/nodeconfig"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// InitSyncDb loads the orakl-config bundle into the in-memory config store.
func InitSyncDb(ctx context.Context) error {
	return sync(ctx)
}

// Sync reloads the orakl-config bundle; fetcher and aggregator pick it up on
// their next refresh.
func Sync(c *fiber.Ctx) error {
	return sync(c.Context())
}

func sync(ctx context.Context) error {
	err := nodeconfig.Default().Sync(ctx)
	if err != nil {
		log.Error().Err(err).Str("Player", "Admin").Str("Url", nodeconfig.URL()).Msg("failed to sync config")
	}
	return err
}

// Get lists the loaded configs with their (process-local) IDs, as used by
// /aggregator/activate/:id and /aggregator/deactivate/:id.
func Get(c *fiber.Ctx) error {
	return c.JSON(nodeconfig.Default().Configs())
}
