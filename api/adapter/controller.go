package adapter

import (
	"encoding/json"
	"fmt"
	"strconv"

	"bisonai.com/orakl/api/feed"
	"bisonai.com/orakl/api/utils"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type AdapterHashModel struct {
	Name     string                           `db:"name" json:"name"`
	Decimals *utils.CustomInt32               `db:"decimals" json:"decimals"`
	Feeds    []feed.FeedWithoutAdapterIdModel `json:"feeds"`
}

type AdapterInsertModel struct {
	AdapterHash string                 `db:"adapter_hash" json:"adapterHash"`
	Name        string                 `db:"name" json:"name" validate:"required"`
	Decimals    *utils.CustomInt32     `db:"decimals" json:"decimals" validate:"required"`
	Feeds       []feed.FeedInsertModel `json:"feeds"`
}

type AdapterModel struct {
	AdapterId   *utils.CustomInt64 `db:"adapter_id" json:"id"`
	AdapterHash string             `db:"adapter_hash" json:"adapterHash"`
	Name        string             `db:"name" json:"name" validate:"required"`
	Decimals    *utils.CustomInt32 `db:"decimals" json:"decimals" validate:"required"`
}

type AdapterDetailModel struct {
	AdapterModel
	Feeds []feed.FeedModel `json:"feeds"`
}

type AdapterIdModel struct {
	AdapterId *utils.CustomInt64 `db:"adapter_id" json:"id"`
}

type FeedIdModel struct {
	FeedId *utils.CustomInt64 `db:"feed_id" json:"id"`
}

func insert(c *fiber.Ctx) error {
	payload := new(AdapterInsertModel)
	if err := c.BodyParser(payload); err != nil {
		panic(err)
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		panic(err)
	}

	err := computeAdapterHash(payload, true)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}

	row, err := utils.QueryRow[AdapterIdModel](c, InsertAdapter, map[string]any{
		"adapter_hash": payload.AdapterHash,
		"name":         payload.Name,
		"decimals":     payload.Decimals})
	if err != nil {
		panic(err)
	}

	for _, item := range payload.Feeds {
		item.AdapterId = row.AdapterId
		_, err := utils.QueryRow[FeedIdModel](c, InsertFeed, map[string]any{
			"name":       item.Name,
			"definition": item.Definition,
			"adapter_id": item.AdapterId})
		if err != nil {
			panic(err)
		}
	}

	result := AdapterModel{AdapterId: row.AdapterId, AdapterHash: payload.AdapterHash, Name: payload.Name, Decimals: payload.Decimals}

	return c.JSON(result)
}

func hash(c *fiber.Ctx) error {
	verifyRaw := c.Query("verify")
	verify, err := strconv.ParseBool(verifyRaw)
	if err != nil {
		panic(err)
	}

	var payload AdapterInsertModel

	if err := c.BodyParser(&payload); err != nil {
		panic(err)
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		panic(err)
	}

	err = computeAdapterHash(&payload, verify)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(payload)
}

func get(c *fiber.Ctx) error {
	results, err := utils.QueryRows[AdapterModel](c, GetAdapter, nil)
	if err != nil {
		panic(err)
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AdapterModel](c, GetAdpaterById, map[string]any{"id": id})
	if err != nil {
		panic(err)
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AdapterModel](c, RemoveAdapter, map[string]any{"id": id})
	if err != nil {
		panic(err)
	}

	return c.JSON(result)
}

func computeAdapterHash(data *AdapterInsertModel, verify bool) error {
	adapterIdRemovedFeeds := make([]feed.FeedWithoutAdapterIdModel, len(data.Feeds))
	for idx, item := range data.Feeds {
		adapterIdRemovedFeeds[idx] = feed.FeedWithoutAdapterIdModel{
			Name:       item.Name,
			Definition: item.Definition,
		}
	}

	input := AdapterHashModel{data.Name, data.Decimals, adapterIdRemovedFeeds}

	out, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to compute adapter hash: %s", err.Error())
	}

	hash := crypto.Keccak256Hash([]byte(out))
	hashString := fmt.Sprintf("0x%x", hash)
	if verify && data.AdapterHash != hashString {
		hashComputeErr := fmt.Errorf("hashes do not match!\nexpected %s, received %s", hashString, data.AdapterHash)
		return fmt.Errorf("failed to compute adapter hash: %s", hashComputeErr.Error())
	}

	data.AdapterHash = hashString
	return nil
}
