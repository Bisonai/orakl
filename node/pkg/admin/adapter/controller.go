package adapter

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"net/http"

	"bisonai.com/orakl/node/pkg/admin/utils"
	"bisonai.com/orakl/node/pkg/bus"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

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
	urls, err := utils.GetOraklConfigUrls(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to get orakl config urls: " + err.Error())
	}

	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			sem <- struct{}{}
			err = processConfigUrl(c.Context(), url)
			if err != nil {
				log.Error().Err(err).Msg("failed to process config url")
			}
			<-sem
		}(url)
	}

	wg.Wait()

	return c.Status(fiber.StatusOK).SendString("syncing request sent")
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

func processConfigUrl(ctx context.Context, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var adapter AdapterInsertModel
	err = json.Unmarshal(body, &adapter)
	if err != nil {
		return err
	}

	validate := validator.New()
	if err = validate.Struct(adapter); err != nil {
		return err
	}

	row, err := db.QueryRow[AdapterModel](ctx, InsertAdapter, map[string]any{
		"name": adapter.Name,
	})
	if err != nil {
		return err
	}

	for _, feed := range adapter.Feeds {
		feed.AdapterId = row.Id
		_, err := db.QueryRow[FeedModel](ctx, InsertFeed, map[string]any{
			"name":       feed.Name,
			"definition": feed.Definition,
			"adapter_id": feed.AdapterId,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
