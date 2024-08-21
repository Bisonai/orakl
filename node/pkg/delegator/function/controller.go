package function

import (
	"encoding/hex"

	"bisonai.com/miko/node/pkg/delegator/utils"
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/klaytn/klaytn/crypto"
)

type FunctionModel struct {
	FunctionId  *utils.CustomInt64 `json:"id" db:"id"`
	Name        string             `json:"name" db:"name"`
	EncodedName string             `json:"encodedName" db:"encodedName"`
	ContractId  *utils.CustomInt64 `json:"contractId" db:"contractId"`
}

type FunctionDetailModel struct {
	FunctionId  *utils.CustomInt64 `json:"id" db:"id"`
	Name        string             `json:"name" db:"name"`
	EncodedName string             `json:"encodedName" db:"encodedName"`
	Address     string             `json:"address" db:"address"`
}

type FunctionInsertModel struct {
	Name       string             `json:"name" db:"name" validate:"required"`
	ContractId *utils.CustomInt64 `json:"contractId" db:"contract_id" validate:"required"`
}

type ContractModel struct {
	Address    string             `json:"address" db:"address"`
	ContractId *utils.CustomInt64 `json:"id" db:"contract_id"`
}

func insert(c *fiber.Ctx) error {
	payload := new(FunctionInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	hash := crypto.Keccak256([]byte(payload.Name))
	encodedName := "0x" + hex.EncodeToString(hash[:4])

	result, err := utils.QueryRow[FunctionModel](c, InsertFunction, map[string]any{
		"name":        payload.Name,
		"contract_id": payload.ContractId,
		"encodedName": encodedName,
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func get(c *fiber.Ctx) error {
	functionResults, err := utils.QueryRows[FunctionModel](c, GetFunction, nil)
	if err != nil {
		return err
	}

	results := make([]FunctionDetailModel, len(functionResults))
	for i, function := range functionResults {
		contract, err := utils.QueryRow[ContractModel](c, GetContractById, map[string]any{"id": function.ContractId})
		if err != nil {
			return err
		}
		results[i] = FunctionDetailModel{
			FunctionId:  function.FunctionId,
			Name:        function.Name,
			EncodedName: function.EncodedName,
			Address:     contract.Address,
		}
	}

	return c.JSON(results)
}

func getById(c *fiber.Ctx) error {
	id := c.Params("id")
	function, err := utils.QueryRow[FunctionModel](c, GetFunctionById, map[string]any{"id": id})
	if err != nil {
		return err
	}
	contract, err := utils.QueryRow[ContractModel](c, GetContractById, map[string]any{"id": function.ContractId})
	if err != nil {
		return err
	}
	result := FunctionDetailModel{
		FunctionId:  function.FunctionId,
		Name:        function.Name,
		EncodedName: function.EncodedName,
		Address:     contract.Address,
	}

	return c.JSON(result)
}

func updateById(c *fiber.Ctx) error {
	id := c.Params("id")
	payload := new(FunctionInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	hash := crypto.Keccak256([]byte(payload.Name))
	encodedName := "0x" + hex.EncodeToString(hash[:4])

	result, err := utils.QueryRow[FunctionModel](c, UpdateFunctionById, map[string]any{"id": id, "name": payload.Name, "contract_id": payload.ContractId, "encodedName": encodedName})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[FunctionModel](c, DeleteFunctionById, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}
