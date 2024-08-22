package reporter

import (
	"strings"

	"bisonai.com/miko/node/pkg/delegator/utils"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
)

type ReporterInsertModel struct {
	Address        string             `json:"address" db:"address" validate:"required"`
	OrganizationId *utils.CustomInt64 `json:"organizationId" db:"organization_id"`
}

type ReporterModel struct {
	ReporterId     *utils.CustomInt64 `db:"id" json:"id"`
	Address        string             `db:"address" json:"address"`
	OrganizationId *utils.CustomInt64 `db:"organization_id" json:"organizationId"`
}

type ReporterDetailModel struct {
	ReporterId       *utils.CustomInt64 `json:"id"`
	Address          string             `json:"address"`
	OrganizationName string             `json:"organization"`
	Contract         []string           `json:"contract"`
}

type ContractModel struct {
	Address    string             `json:"address" db:"address"`
	ContractId *utils.CustomInt64 `json:"id" db:"contract_id"`
}

type OrganizationName struct {
	Name string `json:"name" db:"name"`
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

	result, err := utils.QueryRow[ReporterModel](c, InsertReporter, map[string]any{"address": strings.ToLower(payload.Address), "organizationId": payload.OrganizationId})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	reporters, err := utils.QueryRows[ReporterModel](c, GetReporter, nil)
	if err != nil {
		return err
	}

	results := make([]ReporterDetailModel, len(reporters))
	for i, reporter := range reporters {
		organizationName, err := utils.QueryRow[OrganizationName](c, GetOrganizationName, map[string]any{"organizationId": reporter.OrganizationId})
		if err != nil {
			return err
		}

		contracts, err := utils.QueryRows[ContractModel](c, GetConnectedContracts, map[string]any{"reporterId": reporter.ReporterId})
		if err != nil {
			return err
		}

		contractAddresses := make([]string, len(contracts))
		for j, contract := range contracts {
			contractAddresses[j] = contract.Address
		}

		results[i] = ReporterDetailModel{
			ReporterId:       reporter.ReporterId,
			Address:          reporter.Address,
			OrganizationName: organizationName.Name,
			Contract:         contractAddresses,
		}
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	reporter, err := utils.QueryRow[ReporterModel](c, GetReporterById, map[string]any{"id": id})
	if err != nil {
		return err
	}
	organizationName, err := utils.QueryRow[OrganizationName](c, GetOrganizationName, map[string]any{"organizationId": reporter.OrganizationId})
	if err != nil {
		return err
	}

	contracts, err := utils.QueryRows[ContractModel](c, GetConnectedContracts, map[string]any{"reporterId": reporter.ReporterId})
	if err != nil {
		return err
	}

	contractAddresses := make([]string, len(contracts))
	for i, contract := range contracts {
		contractAddresses[i] = contract.Address
	}

	result := ReporterDetailModel{
		ReporterId:       reporter.ReporterId,
		Address:          reporter.Address,
		OrganizationName: organizationName.Name,
		Contract:         contractAddresses,
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(ReporterInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	result, err := utils.QueryRow[ReporterModel](c, UpdateReporterById, map[string]any{"id": id, "address": payload.Address, "organizationId": payload.OrganizationId})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[ReporterModel](c, DeleteReporterById, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(result)
}
