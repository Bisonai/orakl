package sign

import (
	"strings"

	"bisonai.com/miko/node/pkg/delegator/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// RequireToken accepts only requests with a valid "Authorization: Bearer
// <token>" (SQLite mode). With no tokens configured every request is refused.
func RequireToken(c *fiber.Ctx) error {
	s := utils.GetStore(c)
	if s == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "token store not configured")
	}
	token, ok := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	token = strings.TrimSpace(token)
	if !ok || token == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
	}
	name, valid, err := s.TokenName(c.Context(), token)
	if err != nil {
		return err
	}
	if !valid {
		log.Warn().Str("ip", c.IP()).Str("path", c.Path()).Msg("rejected invalid delegator token")
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	}
	c.Locals("tokenName", name)
	return c.Next()
}

func tokenName(c *fiber.Ctx) string {
	name, _ := c.Locals("tokenName").(string)
	return name
}
