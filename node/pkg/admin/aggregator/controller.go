package aggregator

import (
	"context"
	"fmt"
	"os"
	"sync"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type AggregatorModel struct {
	Id       *int64 `db:"id" json:"id"`
	Name     string `db:"name" json:"name"`
	Active   bool   `db:"active" json:"active"`
	Interval int    `db:"interval" json:"interval"`
}

type AggregatorInsertModel struct {
	Name     string `db:"name" json:"name" validate:"required"`
	Interval *int   `db:"interval" json:"aggregateHeartbeat"`
}

type BulkAggregators struct {
	Aggregators []AggregatorInsertModel `json:"result"`
}

func start(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.START_AGGREGATOR_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start aggregator: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to start aggregator: " + resp.Args["error"].(string))
	}
	return c.SendString("aggregator started")
}

func stop(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.STOP_AGGREGATOR_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop aggregator: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to stop aggregator: " + resp.Args["error"].(string))
	}
	return c.SendString("aggregator stopped")
}

func refresh(c *fiber.Ctx) error {
	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.REFRESH_AGGREGATOR_APP, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh aggregator: " + err.Error())
	}
	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to refresh aggregator: " + resp.Args["error"].(string))
	}
	return c.SendString("aggregator refreshed")
}

func insert(c *fiber.Ctx) error {
	payload := new(AggregatorInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse body for aggregator insert payload: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate aggregator insert payload: " + err.Error())
	}
	result, err := insertAggregator(c.Context(), *payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator insert query: " + err.Error())
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	result, err := db.QueryRows[AggregatorModel](c.Context(), GetAggregator, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator get query: " + err.Error())
	}

	return c.JSON(result)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), GetAggregatorById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator get by id query: " + err.Error())
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), DeleteAggregatorById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator delete by id query: " + err.Error())
	}
	return c.JSON(result)
}

func SyncFromOraklConfig(c *fiber.Ctx) error {
	configUrl := getConfigUrl()

	var aggregators BulkAggregators
	aggregators, err := request.GetRequest[BulkAggregators](configUrl, nil, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get aggregators from config: " + err.Error())
	}

	errs := make(chan error, len(aggregators.Aggregators))
	var wg sync.WaitGroup

	validate := validator.New()
	maxConcurrency := 20
	sem := make(chan struct{}, maxConcurrency)

	for _, aggregator := range aggregators.Aggregators {
		wg.Add(1)
		sem <- struct{}{}
		go func(aggregator AggregatorInsertModel) {
			defer wg.Done()
			defer func() { <-sem }()

			if err = validate.Struct(aggregator); err != nil {
				log.Error().Err(err).Msg("failed to validate orakl config aggregator")
				errs <- err
				return
			}
			_, err := insertAggregator(c.Context(), aggregator)
			if err != nil {
				log.Error().Err(err).Msg("failed to execute aggregator insert query")
				errs <- err
				return
			}
		}(aggregator)
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

func addFromOraklConfig(c *fiber.Ctx) error {
	configUrl := getConfigUrl()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("name is required")
	}

	var aggregators BulkAggregators
	aggregators, err := request.GetRequest[BulkAggregators](configUrl, nil, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config: " + err.Error())
	}

	validate := validator.New()
	for _, aggregator := range aggregators.Aggregators {
		if aggregator.Name == name {
			if err := validate.Struct(aggregator); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("failed to validate orakl config aggregator: " + err.Error())
			}
			result, err := insertAggregator(c.Context(), aggregator)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator insert query: " + err.Error())
			}
			return c.JSON(result)
		}
	}
	return c.Status(fiber.StatusNotFound).SendString("aggregator not found in orakl config")
}

func activate(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), ActivateAggregator, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator activate query: " + err.Error())
	}

	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.ACTIVATE_AGGREGATOR, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send message to aggregator: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to activate aggregator: " + resp.Args["error"].(string))
	}

	return c.JSON(result)
}

func deactivate(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AggregatorModel](c.Context(), DeactivateAggregator, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator deactivate query: " + err.Error())
	}

	msg, err := utils.SendMessage(c, bus.AGGREGATOR, bus.DEACTIVATE_AGGREGATOR, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send message to aggregator: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to deactivate aggregator: " + resp.Args["error"].(string))
	}

	return c.JSON(result)
}

func getConfigUrl() string {
	// TODO: add chain validation (currently only supporting baobab and cypress)
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}
	return fmt.Sprintf("https://config.orakl.network/%s_aggregators.json", chain)
}

func insertAggregator(ctx context.Context, aggregator AggregatorInsertModel) (AggregatorModel, error) {
	if aggregator.Interval == nil {
		newInterval := 5000
		aggregator.Interval = &newInterval
	}

	return db.QueryRow[AggregatorModel](ctx, UpsertAggregatorWithInterval, map[string]any{
		"name":     aggregator.Name,
		"interval": aggregator.Interval,
	})
}
