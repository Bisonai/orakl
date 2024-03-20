package submissionAddress

import (
	"context"
	"fmt"
	"os"
	"sync"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type SubmissionAddressModel struct {
	Id      *int64 `db:"id" json:"id"`
	Name    string `db:"name" json:"name"`
	Address string `db:"address" json:"address"`
}

type SubmissionAddressInsertModel struct {
	Name    string `db:"name" json:"name" validate:"required"`
	Address string `db:"address" json:"address" validate:"required"`
}

type BulkAddresses struct {
	Addresses []SubmissionAddressInsertModel `json:"result"`
}

type AggregatorName struct {
	Name string `json:"name"`
}

func SyncFromOraklConfig(c *fiber.Ctx) error {
	configUrl := getConfigUrl()

	var submissionAddresses BulkAddresses
	submissionAddresses, err := request.GetRequest[BulkAddresses](configUrl, nil, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config: " + err.Error())
	}

	errs := make(chan error, len(submissionAddresses.Addresses))
	var wg sync.WaitGroup

	validate := validator.New()
	maxConcurrency := 20
	sem := make(chan struct{}, maxConcurrency)

	for _, address := range submissionAddresses.Addresses {
		wg.Add(1)
		sem <- struct{}{}
		go func(address SubmissionAddressInsertModel) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := validate.Struct(address); err != nil {
				log.Error().Err(err).Str("Name", address.Name).Str("Address", address.Address).Msg("failed to validate submission address")
				errs <- err
				return
			}

			_, err := insertSubmissionAddress(c.Context(), address)
			if err != nil {
				log.Error().Err(err).Msg("failed to execute submission address insert query")
				errs <- err
				return
			}
		}(address)
	}
	wg.Wait()
	close(errs)

	var errorMessages []string
	for err := range errs {
		errorMessages = append(errorMessages, err.Error())
	}

	if len(errorMessages) > 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(errorMessages)
	}

	return c.Status(fiber.StatusOK).SendString("sync successful")
}

func syncWithAggregator(c *fiber.Ctx) error {
	pairNames, err := db.QueryRows[AggregatorName](c.Context(), GetAggregatorNames, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator sync with adapter query: " + err.Error())
	}

	configUrl := getConfigUrl()

	var submissionAddresses BulkAddresses
	submissionAddresses, err = request.GetRequest[BulkAddresses](configUrl, nil, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config: " + err.Error())
	}

	for _, address := range submissionAddresses.Addresses {
		if containsName(pairNames, address.Name) {
			_, err := insertSubmissionAddress(c.Context(), address)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address insert query: " + err.Error())
			}
		}
	}

	return c.Status(fiber.StatusOK).SendString("sync successful")
}

func addFromOraklConfig(c *fiber.Ctx) error {
	configUrl := getConfigUrl()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("name is required")
	}

	var submissionAddresses BulkAddresses
	submissionAddresses, err := request.GetRequest[BulkAddresses](configUrl, nil, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config: " + err.Error())
	}

	for _, address := range submissionAddresses.Addresses {
		if address.Name == name {
			result, err := insertSubmissionAddress(c.Context(), address)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address insert query: " + err.Error())
			}
			return c.JSON(result)
		}
	}
	return c.Status(fiber.StatusNotFound).SendString("name not found in orakl config")
}

func insert(c *fiber.Ctx) error {
	payload := new(SubmissionAddressInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	result, err := insertSubmissionAddress(c.Context(), *payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address insert query: " + err.Error())
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	result, err := db.QueryRows[SubmissionAddressModel](c.Context(), GetSubmissionAddress, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address get query: " + err.Error())
	}

	return c.JSON(result)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[SubmissionAddressModel](c.Context(), GetSubmissionAddressById, map[string]any{
		"id": id,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address get by id query: " + err.Error())
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[SubmissionAddressModel](c.Context(), DeleteSubmissionAddressById, map[string]any{
		"id": id,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address delete by id query: " + err.Error())
	}
	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(SubmissionAddressInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}

	result, err := db.QueryRow[SubmissionAddressModel](c.Context(), UpdateSubmissionAddressById, map[string]any{
		"id":      id,
		"name":    payload.Name,
		"address": payload.Address,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute submission address update by id query: " + err.Error())
	}

	return c.JSON(result)
}

func insertSubmissionAddress(ctx context.Context, address SubmissionAddressInsertModel) (SubmissionAddressModel, error) {
	result, err := db.QueryRow[SubmissionAddressModel](ctx, UpsertSubmissionAddress, map[string]any{
		"name":    address.Name,
		"address": address.Address,
	})
	if err != nil {
		return SubmissionAddressModel{}, err
	}
	return result, nil
}

func getConfigUrl() string {
	// TODO: add chain validation (currently only supporting baobab and cypress)
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}
	return fmt.Sprintf("https://config.orakl.network/%s_aggregators.json", chain)
}

func containsName(names []AggregatorName, target string) bool {
	for _, name := range names {
		if name.Name == target {
			return true
		}
	}
	return false
}
