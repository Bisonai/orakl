package l2aggregator

import (
	"go-api/chain"
	"go-api/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type l2agregatorPairModel struct {
	Id                  *utils.CustomInt64 `db:"id" json:"id"`
	L1AggregatorAddress string             `db:"l1_aggregator_address" json:"l1AggregatorAddress"`
	L2AggregatorAddress string             `db:"l2_aggregator_address" json:"l2AggregatorAddress"`
	Active              *utils.CustomBool  `db:"active" json:"active"`
	ChainId             *utils.CustomInt64 `db:"chain_id" json:"chainId"`
}

func get(c *fiber.Ctx) error {
	_chain := c.Params("chain")
	l1Address := c.Params("l1Address")

	chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": _chain})
	if err != nil {
		log.Panic(err)
	}

	result, err := utils.QueryRow[l2agregatorPairModel](c, GetL2AggregatorPair, map[string]any{"l1_aggregator_address": l1Address, "chain_id": chain_result.ChainId})
	if err != nil {
		log.Panic(err)
	}

	return c.JSON(result)

}
