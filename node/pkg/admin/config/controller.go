package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/gofiber/fiber/v2"
)

type BulkConfigs struct {
	Configs []ConfigInsertModel `json:"result"`
}

type FeedInsertModel struct {
	Name       string          `db:"name" json:"name" validate:"required"`
	Definition json.RawMessage `db:"definition" json:"definition" validate:"required"`
	ConfigId   *int64          `db:"config_id" json:"configId"`
}

type ConfigInsertModel struct {
	Name              string            `db:"name" json:"name"`
	Address           string            `db:"address" json:"address"`
	FetchInterval     *int              `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval *int              `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval    *int              `db:"submit_interval" json:"submitInterval"`
	Feeds             []FeedInsertModel `json:"feeds"`
}

type ConfigModel struct {
	Id                int64  `db:"id" json:"id"`
	Name              string `db:"name" json:"name"`
	Address           string `db:"address" json:"address"`
	FetchInterval     *int   `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval *int   `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval    *int   `db:"submit_interval" json:"submitInterval"`
}

type ConfigNameIdModel struct {
	Name string `db:"name" json:"name"`
	Id   int64  `db:"id" json:"id"`
}

func Sync(c *fiber.Ctx) error {
	configUrl := getConfigUrl()
	bulkConfigs, err := request.GetRequest[BulkConfigs](configUrl, nil, nil)
	if err != nil {
		return err
	}

	err = bulkUpsertConfigs(c.Context(), bulkConfigs.Configs)
	if err != nil {
		return err
	}

	whereValues := make([]interface{}, 0, len(bulkConfigs.Configs))
	for _, config := range bulkConfigs.Configs {
		whereValues = append(whereValues, config.Name)
	}

	configIds, err := db.BulkSelect[ConfigNameIdModel](c.Context(), "configs", []string{"name", "id"}, []string{"name"}, whereValues)
	if err != nil {
		return err
	}

	configNameIdMap := map[string]int64{}
	for _, configId := range configIds {
		configNameIdMap[configId.Name] = configId.Id
	}

	upsertRows := make([][]any, 0)
	for _, config := range bulkConfigs.Configs {
		for _, feed := range config.Feeds {
			configId, ok := configNameIdMap[config.Name]
			if !ok {
				continue
			}
			upsertRows = append(upsertRows, []any{feed.Name, feed.Definition, configId})
		}
	}

	return db.BulkUpsert(c.Context(), "feeds", []string{"name", "definition", "config_id"}, upsertRows, []string{"name"}, []string{"definition", "config_id"})
}

func Get(c *fiber.Ctx) error {
	configs, err := db.QueryRows[ConfigModel](c.Context(), "SELECT * FROM configs", nil)
	if err != nil {
		return err
	}

	return c.JSON(configs)

}

func getConfigUrl() string {
	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}

	return fmt.Sprintf("https://config.orakl.network/%s_configs.json", chain)
}

func bulkUpsertConfigs(ctx context.Context, configs []ConfigInsertModel) error {
	upsertRows := make([][]any, 0, len(configs))
	for _, config := range configs {
		upsertRows = append(upsertRows, []any{config.Name, config.Address, config.FetchInterval, config.AggregateInterval, config.SubmitInterval})
	}

	return db.BulkUpsert(ctx, "configs", []string{"name", "address", "fetch_interval", "aggregate_interval", "submit_interval"}, upsertRows, []string{"name"}, []string{"address", "fetch_interval", "aggregate_interval", "submit_interval"})
}
