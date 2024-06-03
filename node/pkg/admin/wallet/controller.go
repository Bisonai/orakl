package wallet

import (
	chainUtils "bisonai.com/orakl/node/pkg/chain/utils"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/encryptor"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type WalletModel struct {
	ID *int32 `db:"id" json:"id"`
	Pk string `db:"pk" json:"pk"`
}

type WalletInsertModel struct {
	Pk string `json:"pk" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(WalletInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse request body: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate request body: " + err.Error())
	}

	encryptedPk, err := encryptor.EncryptText(payload.Pk)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to encrypt pk: " + err.Error())
	}

	result, err := db.QueryRow[WalletModel](c.Context(), InsertWallet, map[string]any{
		"pk": encryptedPk})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute insert wallet query: " + err.Error())
	}

	result.Pk = payload.Pk

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := db.QueryRows[WalletModel](c.Context(), GetWallets, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get wallet query: " + err.Error())
	}

	for i, result := range results {
		decryptedPk, err := encryptor.DecryptText(result.Pk)
		if err != nil {
			log.Warn().Err(err).Msg("failed to decrypt pk on get wallets query")
			continue
		}
		results[i].Pk = decryptedPk
	}

	return c.JSON(results)
}

func getAddresses(c *fiber.Ctx) error {
	wallets, err := db.QueryRows[WalletModel](c.Context(), GetWallets, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get wallet query: " + err.Error())
	}
	var result []string
	for _, wallet := range wallets {
		decryptedPk, err := encryptor.DecryptText(wallet.Pk)
		if err != nil {
			log.Warn().Err(err).Msg("failed to decrypt pk on get wallets query")
			continue
		}
		address, err := chainUtils.StringPkToAddressHex(decryptedPk)
		if err != nil {
			log.Warn().Err(err).Msg("failed to convert pk to address on get wallets query")
			continue
		}
		result = append(result, address)
	}

	return c.JSON(result)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[WalletModel](c.Context(), GetWalletById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute get wallet by id query: " + err.Error())
	}
	if result.Pk != "" {
		result.Pk, err = encryptor.DecryptText(result.Pk)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to decrypt pk: " + err.Error())
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(WalletInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse request body: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate request body: " + err.Error())
	}

	encryptedPk, err := encryptor.EncryptText(payload.Pk)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to encrypt pk: " + err.Error())
	}

	result, err := db.QueryRow[WalletModel](c.Context(), UpdateWalletById, map[string]any{"pk": encryptedPk, "id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute update wallet by id query: " + err.Error())
	}

	result.Pk = payload.Pk

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[WalletModel](c.Context(), DeleteWalletById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute delete wallet by id query: " + err.Error())
	}
	return c.JSON(result)
}
