package listener

import (
	"bisonai.com/miko/node/pkg/api/chain"
	"bisonai.com/miko/node/pkg/api/service"
	"bisonai.com/miko/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ListenerUpdateModel struct {
	Address   string `db:"address" json:"address" validate:"required"`
	EventName string `db:"event_name" json:"eventName" validate:"required"`
}

type ListenerSearchModel struct {
	Chain   string `db:"name" json:"chain"`
	Service string `db:"name" json:"service"`
}

type ListenerModel struct {
	ListenerId *utils.CustomInt64 `db:"listener_id" json:"id"`
	Address    string             `db:"address" json:"address" validate:"required"`
	EventName  string             `db:"event_name" json:"eventName" validate:"required"`
	Service    string             `db:"service_name" json:"service" validate:"required"`
	Chain      string             `db:"chain_name" json:"chain" validate:"required"`
}

type ListenerInsertModel struct {
	Address   string `db:"address" json:"address" validate:"required"`
	EventName string `db:"event_name" json:"eventName" validate:"required"`
	Service   string `db:"service_name" json:"service" validate:"required"`
	Chain     string `db:"chain_name" json:"chain" validate:"required"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ListenerInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": payload.Chain})
	if err != nil {
		return err
	}

	service_result, err := utils.QueryRow[service.ServiceModel](c, service.GetServiceByName, map[string]any{"name": payload.Service})
	if err != nil {
		return err
	}

	result, err := utils.QueryRow[ListenerModel](c, InsertListener, map[string]any{
		"address":    payload.Address,
		"event_name": payload.EventName,
		"chain_id":   chain_result.ChainId,
		"service_id": service_result.ServiceId})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	payload := new(ListenerSearchModel)
	params := GetListenerQueryParams{}

	if len(c.Body()) == 0 {
		results, err := utils.QueryRows[ListenerModel](c, GenerateGetListenerQuery(params), nil)
		if err != nil {
			return err
		}

		return c.JSON(results)
	}

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	if payload.Chain != "" {
		chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": payload.Chain})
		if err != nil {
			return err
		}
		params.ChainId = chain_result.ChainId.String()
	}

	if payload.Service != "" {
		service_result, err := utils.QueryRow[service.ServiceModel](c, service.GetServiceByName, map[string]any{"name": payload.Service})
		if err != nil {
			return err
		}
		params.ServiceId = service_result.ServiceId.String()
	}

	results, err := utils.QueryRows[ListenerModel](c, GenerateGetListenerQuery(params), nil)
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ListenerModel](c, GetListenerById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ListenerUpdateModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ListenerModel](c, UpdateListenerById, map[string]any{
		"id":         id,
		"address":    payload.Address,
		"event_name": payload.EventName})
	if err != nil {
		return err
	}

	return c.JSON(result)

}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ListenerModel](c, DeleteListenerById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
