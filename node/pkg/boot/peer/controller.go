package peer

import (
	"bisonai.com/miko/node/pkg/boot/utils/peerstore"
	libp2pUtils "bisonai.com/miko/node/pkg/libp2p/utils"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

type PeerModel = peerstore.Peer

type PeerInsertModel struct {
	Url string `db:"url" json:"url" validate:"required"`
}

// syncHandler gets the store by closure, not c.Locals: fasthttp calls Close()
// on io.Closer user values at the end of every request.
func syncHandler(store *peerstore.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return sync(c, store)
	}
}

func sync(c *fiber.Ctx, store *peerstore.Store) error {
	payload := new(PeerInsertModel)
	if err := c.BodyParser(payload); err != nil {
		log.Error().Err(err).Msg("Failed to parse request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to parse request")
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		log.Error().Any("payload", payload).Err(err).Msg("Failed to validate request")
		return c.Status(fiber.StatusBadRequest).SendString("Failed to validate request")
	}

	h, ok := c.Locals("host").(*host.Host)
	if !ok {
		log.Error().Msg("Failed to get host")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get host")
	}

	isAlive, err := libp2pUtils.IsHostAlive(c.Context(), *h, payload.Url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check peer")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to check peer")
	}
	if !isAlive {
		log.Info().Str("peer", payload.Url).Msg("invalid peer")
		return c.Status(fiber.StatusBadRequest).SendString("invalid peer")
	}

	// peers known before this one registers; the caller does not get itself back unless already stored
	peers, err := store.List(c.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute get query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute get query")
	}

	_, _, err = store.Insert(c.Context(), payload.Url)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute insert query")
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to execute insert query")
	}

	return c.JSON(peers)
}
