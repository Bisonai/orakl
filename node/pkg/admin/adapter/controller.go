package adapter

import (
	"encoding/json"
	"sync"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	oraklUtil "bisonai.com/orakl/node/pkg/utils"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type BulkAdapters struct {
	Adapters []AdapterInsertModel `json:"adapters"`
}

type AdapterModel struct {
	Id     *int64 `db:"id" json:"id"`
	Name   string `db:"name" json:"name"`
	Active bool   `db:"active" json:"active"`
}

type FeedModel struct {
	Id         *int64          `db:"id" json:"id"`
	Name       string          `db:"name" json:"name"`
	Definition json.RawMessage `db:"definition" json:"definition"`
	AdapterId  *int64          `db:"adapter_id" json:"adapterId"`
}

type FeedInsertModel struct {
	Name       string          `db:"name" json:"name" validate:"required"`
	Definition json.RawMessage `db:"definition" json:"definition" validate:"required"`
	AdapterId  *int64          `db:"adapter_id" json:"adapterId"`
}

type AdapterInsertModel struct {
	Name  string            `db:"name" json:"name" validate:"required"`
	Feeds []FeedInsertModel `json:"feeds"`
}

type AdapterDetailModel struct {
	AdapterModel
	Feeds []FeedModel `json:"feeds"`
}

func syncFromOraklConfig(c *fiber.Ctx) error {
	var adapters BulkAdapters
	adapters, err := oraklUtil.GetRequest[BulkAdapters]("https://config.orakl.network/adapters.json", nil, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config: " + err.Error())
	}

	errs := make(chan error, len(adapters.Adapters))
	var wg sync.WaitGroup

	maxConcurrency := 20
	sem := make(chan struct{}, maxConcurrency)

	for _, adapter := range adapters.Adapters {
		wg.Add(1)
		sem <- struct{}{}
		go func(adapter AdapterInsertModel) {
			defer wg.Done()
			defer func() { <-sem }()
			validate := validator.New()
			if err = validate.Struct(adapter); err != nil {
				log.Error().Err(err).Msg("failed to validate orakl config adapter")
				errs <- err
				return
			}

			row, err := db.QueryRow[AdapterModel](c.Context(), UpsertAdapter, map[string]any{
				"name": adapter.Name,
			})
			if err != nil {
				log.Error().Err(err).Msg("failed to execute adapter insert query")
				errs <- err
				return
			}

			for _, feed := range adapter.Feeds {
				feed.AdapterId = row.Id
				_, err := db.QueryRow[FeedModel](c.Context(), UpsertFeed, map[string]any{
					"name":       feed.Name,
					"definition": feed.Definition,
					"adapter_id": feed.AdapterId,
				})
				if err != nil {
					log.Error().Err(err).Msg("failed to execute feed insert query")
					errs <- err
					continue
				}
			}
		}(adapter)
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

	return c.Status(fiber.StatusOK).SendString("syncing request sent")
}

func addFromOraklConfig(c *fiber.Ctx) error {
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).SendString("name is required")
	}

	var adapters BulkAdapters
	adapters, err := oraklUtil.GetRequest[BulkAdapters]("https://config.orakl.network/adapters.json", nil, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config: " + err.Error())
	}

	for _, adapter := range adapters.Adapters {
		if adapter.Name == name {
			validate := validator.New()
			if err = validate.Struct(adapter); err != nil {
				log.Error().Err(err).Msg("failed to validate orakl config adapter")
				return c.Status(fiber.StatusInternalServerError).SendString("failed to validate orakl config adapter: " + err.Error())
			}

			row, err := db.QueryRow[AdapterModel](c.Context(), UpsertAdapter, map[string]any{
				"name": adapter.Name,
			})
			if err != nil {
				log.Error().Err(err).Msg("failed to execute adapter insert query")
				return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter insert query: " + err.Error())
			}

			for _, feed := range adapter.Feeds {
				feed.AdapterId = row.Id
				_, err := db.QueryRow[FeedModel](c.Context(), UpsertFeed, map[string]any{
					"name":       feed.Name,
					"definition": feed.Definition,
					"adapter_id": feed.AdapterId,
				})
				if err != nil {
					log.Error().Err(err).Msg("failed to execute feed insert query")
					return c.Status(fiber.StatusInternalServerError).SendString("failed to execute feed insert query: " + err.Error())
				}
			}

			result := AdapterModel{Id: row.Id, Name: row.Name, Active: row.Active}
			return c.JSON(result)
		}
	}
	return c.Status(fiber.StatusInternalServerError).SendString("adapter not found in orakl config")
}

func insert(c *fiber.Ctx) error {
	payload := new(AdapterInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to parse body for adapter insert payload: " + err.Error())
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to validate adapter insert payload: " + err.Error())
	}

	row, err := db.QueryRow[AdapterModel](c.Context(), InsertAdapter, map[string]any{
		"name": payload.Name,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter insert query: " + err.Error())
	}

	for _, feed := range payload.Feeds {
		feed.AdapterId = row.Id
		_, err := db.QueryRow[FeedModel](c.Context(), InsertFeed, map[string]any{
			"name":       feed.Name,
			"definition": feed.Definition,
			"adapter_id": feed.AdapterId,
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("failed to execute feed insert query: " + err.Error())
		}
	}

	result := AdapterModel{Id: row.Id, Name: row.Name, Active: row.Active}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	results, err := db.QueryRows[AdapterModel](c.Context(), GetAdapter, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter get query: " + err.Error())
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AdapterModel](c.Context(), GetAdapterById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter get by id query: " + err.Error())
	}
	return c.JSON(result)
}

func getDetailById(c *fiber.Ctx) error {
	id := c.Params("id")
	adapter, err := db.QueryRow[AdapterModel](c.Context(), GetAdapterById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter get by id query: " + err.Error())
	}
	feeds, err := db.QueryRows[FeedModel](c.Context(), GetFeedsByAdapterId, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute feed get by adapter id query: " + err.Error())
	}
	result := AdapterDetailModel{AdapterModel: adapter, Feeds: feeds}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AdapterModel](c.Context(), DeleteAdapterById, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter delete by id query: " + err.Error())
	}
	return c.JSON(result)
}

func activate(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AdapterModel](c.Context(), ActivateAdapter, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter activate query: " + err.Error())
	}

	msg, err := utils.SendMessage(c, bus.FETCHER, bus.ACTIVATE_FETCHER, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send activate message to fetcher: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to activate adapter: " + resp.Args["error"].(string))
	}

	return c.JSON(result)
}

func deactivate(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := db.QueryRow[AdapterModel](c.Context(), DeactivateAdapter, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to execute adapter deactivate query: " + err.Error())
	}

	msg, err := utils.SendMessage(c, bus.FETCHER, bus.DEACTIVATE_FETCHER, map[string]any{"id": id})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to send deactivate message to fetcher: " + err.Error())
	}

	resp := <-msg.Response
	if !resp.Success {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to deactivate adapter: " + resp.Args["error"].(string))
	}

	return c.JSON(result)
}
