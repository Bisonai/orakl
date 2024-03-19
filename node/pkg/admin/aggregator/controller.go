package aggregator

import (
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
	Id     *int64 `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Active bool   `db:"active" json:"active"`
}

type AggregatorInsertModel struct {
	Name string `db:"name" json:"name" validate:"required"`
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

	result, err := db.QueryRow[AggregatorModel](c.Context(), InsertAggregator, map[string]any{
		"name": payload.Name,
	})
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

func syncWithAdapter(c *fiber.Ctx) error {
	result, err := db.QueryRows[AggregatorModel](c.Context(), SyncAggregator, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute aggregator sync with adapter query: " + err.Error())
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

			_, err := db.QueryRow[AggregatorModel](c.Context(), UpsertAggregator, map[string]any{
				"name": aggregator.Name,
			})
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
	chain := os.Getenv("CHAIN")
	if chain == "" {
		return "https://config.orakl.network/baobab_aggregators.json"
	}
	return fmt.Sprintf("https://config.orakl.network/%s_aggregators.json", chain)
}
