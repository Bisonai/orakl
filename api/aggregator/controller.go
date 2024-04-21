package aggregator

import (
	"encoding/json"
	"fmt"
	"strconv"

	"bisonai.com/orakl/api/adapter"
	"bisonai.com/orakl/api/chain"
	"bisonai.com/orakl/api/feed"
	"bisonai.com/orakl/api/utils"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type WrappedUpdateModel struct {
	Data AggregatorUpdateModel `json:"data"`
}

type AggregatorUpdateModel struct {
	Active *utils.CustomBool `db:"active" json:"active"`
	Chain  string            `db:"chain" json:"chain" validate:"required"`
}

type AggregatorInsertModel struct {
	AggregatorHash    string             `db:"aggregator_hash" json:"aggregatorHash"`
	Active            *utils.CustomBool  `db:"active" json:"active"`
	Name              string             `db:"name" json:"name" validate:"required"`
	Address           string             `db:"address" json:"address" validate:"required"`
	Heartbeat         *utils.CustomInt32 `db:"heartbeat" json:"heartbeat" validate:"required"`
	Threshold         *utils.CustomFloat `db:"threshold" json:"threshold" validate:"required"`
	AbsoluteThreshold *utils.CustomFloat `db:"absolute_threshold" json:"absoluteThreshold" validate:"required"`
	AdapterHash       string             `db:"adapter_hash" json:"adapterHash" validate:"required"`
	Chain             string             `db:"chain" json:"chain" validate:"required"`
	FetcherType       *utils.CustomInt32 `db:"fetcher_type" json:"fetcherType"`
}

type AggregatorResultModel struct {
	AggregatorId      *utils.CustomInt64 `db:"aggregator_id" json:"id"`
	AggregatorHash    string             `db:"aggregator_hash" json:"aggregatorHash"`
	Active            *utils.CustomBool  `db:"active" json:"active"`
	Name              string             `db:"name" json:"name"`
	Address           string             `db:"address" json:"address"`
	Heartbeat         *utils.CustomInt32 `db:"heartbeat" json:"heartbeat"`
	Threshold         *utils.CustomFloat `db:"threshold" json:"threshold"`
	AbsoluteThreshold *utils.CustomFloat `db:"absolute_threshold" json:"absoluteThreshold"`
	AdapterId         *utils.CustomInt64 `db:"adapter_id" json:"adapterId"`
	ChainId           *utils.CustomInt64 `db:"chain_id" json:"chainId"`
	FetcherType       *utils.CustomInt32 `db:"fetcher_type" json:"fetcherType"`
}

type AggregatorDetailResultModel struct {
	AggregatorResultModel
	Adapter adapter.AdapterDetailModel `json:"adapter"`
}

type _AggregatorInsertModel struct {
	AggregatorHash    string             `db:"aggregator_hash" json:"aggregatorHash"`
	Active            *utils.CustomBool  `db:"active" json:"active"`
	Name              string             `db:"name" json:"name"`
	Address           string             `db:"address" json:"address"`
	Heartbeat         *utils.CustomInt32 `db:"heartbeat" json:"heartbeat"`
	Threshold         *utils.CustomFloat `db:"threshold" json:"threshold"`
	AbsoluteThreshold *utils.CustomFloat `db:"absolute_threshold" json:"absoluteThreshold"`
	AdapterId         *utils.CustomInt64 `db:"adapter_id" json:"adapterId"`
	ChainId           *utils.CustomInt64 `db:"chain_id" json:"chainId"`
	FetcherType       *utils.CustomInt32 `db:"fetcher_type" json:"fetcherType"`
}

type AggregatorHashComputeProcessModel struct {
	Name              string             `db:"name" json:"name"`
	Heartbeat         *utils.CustomInt32 `db:"heartbeat" json:"heartbeat"`
	Threshold         *utils.CustomFloat `db:"threshold" json:"threshold"`
	AbsoluteThreshold *utils.CustomFloat `db:"absolute_threshold" json:"absoluteThreshold"`
	AdapterHash       string             `db:"adapter_hash" json:"adapterHash"`
}

type AggregatorHashComputeInputModel struct {
	AggregatorHash string `db:"aggregator_hash" json:"aggregatorHash"`
	AggregatorHashComputeProcessModel
}

type AggregatorIdModel struct {
	AggregatorId *utils.CustomInt64 `db:"aggregator_id" json:"id"`
}

func insert(c *fiber.Ctx) error {
	payload := new(AggregatorInsertModel)
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

	adapter_result, err := utils.QueryRow[adapter.AdapterModel](c, adapter.GetAdapterByHash, map[string]any{"adapter_hash": payload.AdapterHash})
	if err != nil {
		return err
	}

	hashComputeParam := AggregatorHashComputeInputModel{
		AggregatorHash: payload.AggregatorHash,
		AggregatorHashComputeProcessModel: AggregatorHashComputeProcessModel{
			Name:              payload.Name,
			Heartbeat:         payload.Heartbeat,
			Threshold:         payload.Threshold,
			AbsoluteThreshold: payload.AbsoluteThreshold,
			AdapterHash:       payload.AdapterHash,
		},
	}
	err = computeAggregatorHash(&hashComputeParam, true)
	if err != nil {
		return err
	}

	insertParam := _AggregatorInsertModel{
		AggregatorHash:    payload.AggregatorHash,
		Active:            payload.Active,
		Name:              payload.Name,
		Address:           payload.Address,
		Heartbeat:         payload.Heartbeat,
		Threshold:         payload.Threshold,
		AbsoluteThreshold: payload.AbsoluteThreshold,
		AdapterId:         adapter_result.AdapterId,
		ChainId:           chain_result.ChainId,
		FetcherType:       payload.FetcherType,
	}

	if insertParam.Active == nil {
		insertBool := utils.CustomBool(false)
		insertParam.Active = &insertBool
	}

	if insertParam.FetcherType == nil {
		insertFetcherType := utils.CustomInt32(0)
		insertParam.FetcherType = &insertFetcherType
	}

	row, err := utils.QueryRow[AggregatorIdModel](c, InsertAggregator, map[string]any{
		"aggregator_hash":    insertParam.AggregatorHash,
		"active":             insertParam.Active,
		"name":               insertParam.Name,
		"address":            insertParam.Address,
		"heartbeat":          insertParam.Heartbeat,
		"threshold":          insertParam.Threshold,
		"absolute_threshold": insertParam.AbsoluteThreshold,
		"adapter_id":         insertParam.AdapterId,
		"chain_id":           insertParam.ChainId,
		"fetcher_type":       insertParam.FetcherType,
	})
	if err != nil {
		return err
	}

	result := AggregatorResultModel{
		AggregatorId:      row.AggregatorId,
		AggregatorHash:    insertParam.AggregatorHash,
		Active:            insertParam.Active,
		Name:              insertParam.Name,
		Address:           insertParam.Address,
		Heartbeat:         insertParam.Heartbeat,
		Threshold:         insertParam.Threshold,
		AbsoluteThreshold: insertParam.AbsoluteThreshold,
		AdapterId:         insertParam.AdapterId,
		ChainId:           insertParam.ChainId,
		FetcherType:       insertParam.FetcherType,
	}

	return c.JSON(result)
}

func hash(c *fiber.Ctx) error {
	verifyRaw := c.Query("verify")
	verify, err := strconv.ParseBool(verifyRaw)
	if err != nil {
		return err
	}

	payload := new(AggregatorInsertModel)
	if err := c.BodyParser(payload); err != nil {
		return err
	}

	validate := validator.New()
	if err := validate.StructExcept(payload, "Chain"); err != nil {
		return err
	}

	hashComputeParam := AggregatorHashComputeInputModel{
		AggregatorHash: payload.AggregatorHash,
		AggregatorHashComputeProcessModel: AggregatorHashComputeProcessModel{
			Name:              payload.Name,
			Heartbeat:         payload.Heartbeat,
			Threshold:         payload.Threshold,
			AbsoluteThreshold: payload.AbsoluteThreshold,
			AdapterHash:       payload.AdapterHash,
		},
	}

	err = computeAggregatorHash(&hashComputeParam, verify)
	if err != nil {
		return err
	}

	return c.JSON(hashComputeParam)
}

func get(c *fiber.Ctx) error {
	queries := c.Queries()
	queryParam := GetAggregatorQueryParams{
		Active:  queries["active"],
		Chain:   queries["chain"],
		Address: queries["address"],
	}
	queryString, err := GenerateGetAggregatorQuery(queryParam)
	if err != nil {
		return err
	}

	results, err := utils.QueryRows[AggregatorResultModel](c, queryString, nil)
	if err != nil {
		return err
	}

	return c.JSON(results)
}

func getByHashAndChain(c *fiber.Ctx) error {
	var result = new(AggregatorDetailResultModel)
	hash := c.Params("hash")
	_chain := c.Params("chain")

	chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": _chain})
	if err != nil {
		return err
	}

	result.AggregatorResultModel, err = utils.QueryRow[AggregatorResultModel](c, GetAggregatorByChainAndHash, map[string]any{
		"aggregator_hash": hash,
		"chain_id":        chain_result.ChainId,
	})
	if err != nil {
		return err
	}

	result.Adapter.AdapterModel, err = utils.QueryRow[adapter.AdapterModel](c, adapter.GetAdpaterById, map[string]any{"id": result.AggregatorResultModel.AdapterId})
	if err != nil {
		return err
	}

	result.Adapter.Feeds, err = utils.QueryRows[feed.FeedModel](c, feed.GetFeedsByAdapterId, map[string]any{"id": result.AggregatorResultModel.AdapterId})
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func deleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := utils.QueryRow[AggregatorResultModel](c, RemoveAggregator, map[string]any{"id": id})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func updateByHash(c *fiber.Ctx) error {
	hash := c.Params("hash")
	_payload := new(WrappedUpdateModel)
	if err := c.BodyParser(_payload); err != nil {
		return err
	}

	payload := _payload.Data

	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		return err
	}

	if payload.Active == nil {
		insertBool := utils.CustomBool(false)
		payload.Active = &insertBool
	}

	chain_result, err := utils.QueryRow[chain.ChainModel](c, chain.GetChainByName, map[string]any{"name": payload.Chain})
	if err != nil {
		return err
	}

	result, err := utils.QueryRow[AggregatorResultModel](c, UpdateAggregatorByHash, map[string]any{
		"active":   payload.Active,
		"hash":     hash,
		"chain_id": chain_result.ChainId})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

func computeAggregatorHash(data *AggregatorHashComputeInputModel, verify bool) error {
	input := data
	processData := input.AggregatorHashComputeProcessModel
	out, err := json.Marshal(processData)
	if err != nil {
		return fmt.Errorf("failed to compute adapter hash: %s", err.Error())
	}

	hash := crypto.Keccak256Hash([]byte(out))
	hashString := fmt.Sprintf("0x%x", hash)
	if verify && data.AggregatorHash != hashString {
		hashComputeErr := fmt.Errorf("hashes do not match!\nexpected %s, received %s", hashString, data.AggregatorHash)
		return fmt.Errorf("failed to compute adapter hash: %s", hashComputeErr.Error())
	}

	data.AggregatorHash = hashString
	return nil
}
