package reporter

import (
	"bisonai.com/miko/node/pkg/api/chain"
	"bisonai.com/miko/node/pkg/api/service"
	"bisonai.com/miko/node/pkg/api/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ReporterUpdateModel struct {
	Address       string `db:"address" json:"address"  validate:"required"`
	PrivateKey    string `db:"privateKey" json:"privateKey"  validate:"required"`
	OracleAddress string `db:"oracleAddress" json:"oracleAddress"  validate:"required"`
}

type ReporterModel struct {
	ReporterId    *utils.CustomInt64 `db:"reporter_id" json:"id"`
	Address       string             `db:"address" json:"address" validate:"required"`
	PrivateKey    string             `db:"privateKey" json:"privateKey" validate:"required"`
	OracleAddress string             `db:"oracleAddress" json:"oracleAddress" validate:"required"`
	Service       string             `db:"service_name" json:"service" validate:"required"`
	Chain         string             `db:"chain_name" json:"chain" validate:"required"`
}

type ReporterInsertModel struct {
	Address       string `db:"address" json:"address" validate:"required"`
	PrivateKey    string `db:"privateKey" json:"privateKey" validate:"required"`
	OracleAddress string `db:"oracleAddress" json:"oracleAddress" validate:"required"`
	Service       string `db:"service_name" json:"service" validate:"required"`
	Chain         string `db:"chain_name" json:"chain" validate:"required"`
}

type ReporterSearchModel struct {
	Chain   string `db:"chain_name" json:"chain"`
	Service string `db:"service_name" json:"service"`
}

type ReporterSearchByOracleAddressModel struct {
	OracleAddress string `db:"oracleAddress" json:"oracleAddress" validate:"required"`
	Chain         string `db:"chain_name" json:"chain"`
	Service       string `db:"service_name" json:"service"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ReporterInsertModel)
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

	encrypted, err := utils.EncryptText(payload.PrivateKey)
	if err != nil {
		return err
	}

	result, err := utils.QueryRow[ReporterModel](c, InsertReporter, map[string]any{
		"address":       payload.Address,
		"privateKey":    encrypted,
		"oracleAddress": payload.OracleAddress,
		"chain_id":      chain_result.ChainId,
		"service_id":    service_result.ServiceId})
	if err != nil {
		return err
	}

	result.PrivateKey = payload.PrivateKey

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	payload := new(ReporterSearchModel)
	params := GetReporterQueryParams{}

	if len(c.Body()) == 0 {
		results, err := utils.QueryRows[ReporterModel](c, GenerateGetReporterQuery(params), nil)
		if err != nil {
			return err
		}
		for i := range results {
			decrypted, err := utils.DecryptText(results[i].PrivateKey)
			if err != nil {
				return err
			}
			results[i].PrivateKey = decrypted
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

	results, err := utils.QueryRows[ReporterModel](c, GenerateGetReporterQuery(params), nil)
	if err != nil {
		return err
	}

	for i := range results {
		decrypted, err := utils.DecryptText(results[i].PrivateKey)
		if err != nil {
			return err
		}
		results[i].PrivateKey = decrypted
	}

	return c.JSON(results)
}

func getByOracleAddress(c *fiber.Ctx) error {
	oracleAddress := c.Params("oracleAddress")
	payload := new(ReporterSearchModel)
	params := GetReporterQueryParams{}

	if err := c.BodyParser(payload); err != nil {
		return err
	}

	params.OracleAddress = oracleAddress

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

	results, err := utils.QueryRows[ReporterModel](c, GenerateGetReporterQuery(params), nil)
	if err != nil {
		return err
	}

	for i := range results {
		decrypted, err := utils.DecryptText(results[i].PrivateKey)
		if err != nil {
			return err
		}
		results[i].PrivateKey = decrypted
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ReporterModel](c, GetReporterById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	decrypted, err := utils.DecryptText(result.PrivateKey)
	if err != nil {
		return err
	}
	result.PrivateKey = decrypted

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ReporterUpdateModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	encrypted, err := utils.EncryptText(payload.PrivateKey)
	if err != nil {
		return err
	}

	result, err := utils.QueryRow[ReporterModel](c, UpdateReporterById, map[string]any{
		"id":            id,
		"address":       payload.Address,
		"privateKey":    encrypted,
		"oracleAddress": payload.OracleAddress})

	if err != nil {
		return err
	}

	result.PrivateKey = payload.PrivateKey

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ReporterModel](c, DeleteReporterById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	decrypted, err := utils.DecryptText(result.PrivateKey)
	if err != nil {
		return err
	}
	result.PrivateKey = decrypted

	return c.JSON(result)
}
