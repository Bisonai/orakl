package contract

import (
	"strings"

	"bisonai.com/orakl/node/pkg/delegator/utils"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
)

type ContractInsertModel struct {
	Address string `json:"address" db:"address" validate:"required"`
}

type ContractModel struct {
	Address    string             `json:"address" db:"address"`
	ContractId *utils.CustomInt64 `json:"id" db:"contract_id"`
}

type ContractDetailModel struct {
	ContractId  *utils.CustomInt64 `json:"id"`
	Address     string             `json:"address"`
	Reporter    []string           `json:"reporter"`
	EncodedName []string           `json:"encodedName"`
}

type ContractConnectModel struct {
	ContractId *utils.CustomInt64 `json:"contractId" validate:"required" db:"A"`
	ReporterId *utils.CustomInt64 `json:"reporterId" validate:"required" db:"B"`
}

type FunctionModel struct {
	FunctionId  *utils.CustomInt64 `json:"id" db:"id"`
	Name        string             `json:"name" db:"name"`
	EncodedName string             `json:"encodedName" db:"encodedName"`
	ContractId  *utils.CustomInt64 `json:"contractId" db:"contractId"`
}

type ReporterModel struct {
	ReporterId     *utils.CustomInt64 `json:"id" db:"id"`
	Address        string             `json:"address" db:"address" `
	OrganizationId *utils.CustomInt64 `json:"organizationId" db:"organization_id"`
}

func insert(c *fiber.Ctx) error {
	payload := new(ContractInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ContractModel](c, InsertContract, map[string]any{"address": strings.ToLower(payload.Address)})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {

	contractResults, err := utils.QueryRows[ContractModel](c, GetContract, nil)
	if err != nil {
		return err
	}

	results := make([]ContractDetailModel, len(contractResults))
	for i, contract := range contractResults {
		results[i].Address = contract.Address
		results[i].ContractId = contract.ContractId

		reporters, err := utils.QueryRows[ReporterModel](c, GetConnectedReporters, map[string]any{"contractId": contract.ContractId})
		if err != nil {
			return err
		}

		results[i].Reporter = make([]string, len(reporters))
		for j, reporter := range reporters {
			results[i].Reporter[j] = reporter.Address
		}

		functions, err := utils.QueryRows[FunctionModel](c, GetConnectedFunctions, map[string]any{"contractId": contract.ContractId})
		if err != nil {
			return err
		}

		results[i].EncodedName = make([]string, len(functions))
		for j, function := range functions {
			results[i].EncodedName[j] = function.EncodedName
		}
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	contractResult, err := utils.QueryRow[ContractModel](c, GetContractById, map[string]any{"id": id})
	if err != nil {
		return err
	}
	result := ContractDetailModel{
		Address:    contractResult.Address,
		ContractId: contractResult.ContractId,
	}
	reporters, err := utils.QueryRows[ReporterModel](c, GetConnectedReporters, map[string]any{"contractId": contractResult.ContractId})
	if err != nil {
		return err
	}
	result.Reporter = make([]string, len(reporters))
	for i, reporter := range reporters {
		result.Reporter[i] = reporter.Address
	}

	functions, err := utils.QueryRows[FunctionModel](c, GetConnectedFunctions, map[string]any{"contractId": contractResult.ContractId})
	if err != nil {
		return err
	}

	result.EncodedName = make([]string, len(functions))
	for i, function := range functions {
		result.EncodedName[i] = function.EncodedName
	}

	return c.JSON(result)
}

func connectReporter(c *fiber.Ctx) error {
	payload := new(ContractConnectModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	err := utils.RawQueryWithoutReturn(c, ConnectReporter, map[string]any{"contractId": payload.ContractId, "reporterId": payload.ReporterId})
	if err != nil {
		return err
	}

	return c.JSON(nil)
}

func disconnectReporter(c *fiber.Ctx) error {
	payload := new(ContractConnectModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	err := utils.RawQueryWithoutReturn(c, DisconnectReporter, map[string]any{"contractId": payload.ContractId, "reporterId": payload.ReporterId})
	if err != nil {
		return err
	}
	return c.JSON(nil)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ContractInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ContractModel](c, UpdateContract, map[string]any{"id": id, "address": payload.Address})
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ContractModel](c, DeleteContract, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(result)
}
