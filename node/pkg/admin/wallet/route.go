package wallet

import (
	"github.com/gofiber/fiber/v2"
)

func Routes(router fiber.Router) {
	wallets := router.Group("/wallet")

	wallets.Post("", insert)
	wallets.Get("", get)
	wallets.Get("/addresses", getAddresses)
	wallets.Get("/signer", getSignerAddress)
	wallets.Get("/:id", getById)
	wallets.Patch("/:id", updateById)
	wallets.Delete("/:id", deleteById)
}
