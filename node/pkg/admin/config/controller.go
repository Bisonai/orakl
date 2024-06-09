package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"bisonai.com/orakl/node/pkg/admin/feed"
	"bisonai.com/orakl/node/pkg/db"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type BulkConfigs struct {
	Configs []ConfigInsertModel `json:"result"`
}

type FeedInsertModel struct {
	Name       string          `db:"name" json:"name" validate:"required"`
	Definition json.RawMessage `db:"definition" json:"definition" validate:"required"`
	ConfigId   *int32          `db:"config_id" json:"configId"`
}

type ConfigInsertModel struct {
	Name              string            `db:"name" json:"name"`
	FetchInterval     *int              `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval *int              `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval    *int              `db:"submit_interval" json:"submitInterval"`
	Feeds             []FeedInsertModel `json:"feeds"`
}

type ConfigModel struct {
	ID                int32  `db:"id" json:"id"`
	Name              string `db:"name" json:"name"`
	FetchInterval     *int   `db:"fetch_interval" json:"fetchInterval"`
	AggregateInterval *int   `db:"aggregate_interval" json:"aggregateInterval"`
	SubmitInterval    *int   `db:"submit_interval" json:"submitInterval"`
}

type ConfigNameIdModel struct {
	Name string `db:"name" json:"name"`
	ID   int32  `db:"id" json:"id"`
}

func InitSyncDb(ctx context.Context) error {
	return sync(ctx)
}

func Sync(c *fiber.Ctx) error {
	return sync(c.Context())
}

func sync(ctx context.Context) error {
	configUrl := getConfigUrl()
	loadedConfigs, err := request.GetRequest[[]ConfigInsertModel](configUrl, nil, nil)
	if err != nil {
		return err
	}
	loadedConfigMap := map[string]ConfigInsertModel{}
	loadedFeedMap := map[string]FeedInsertModel{}
	for _, config := range loadedConfigs {
		loadedConfigMap[config.Name] = config
		for _, feed := range config.Feeds {
			loadedFeedMap[feed.Name] = feed
		}
	}

	// remove invalid configs
	dbConfigs, err := db.QueryRows[ConfigModel](ctx, SelectConfigQuery, nil)
	if err != nil {
		return err
	}

	removingConfigs := []string{}
	for _, dbConfig := range dbConfigs {
		_, ok := loadedConfigMap[dbConfig.Name]
		if !ok {
			log.Info().Str("Player", "Config").Str("Config", dbConfig.Name).Msg("Config not found in config")
			removingConfigs = append(removingConfigs, strconv.Itoa(int(dbConfig.ID)))
		}
	}
	err = db.QueryWithoutResult(ctx, BulkDeleteConfigQuery, map[string]any{"ids": strings.Join(removingConfigs, ",")})
	if err != nil {
		return err
	}

	// remove invalid feeds
	dbFeeds, err := db.QueryRows[feed.FeedModel](ctx, feed.GetFeed, nil)
	if err != nil {
		return err
	}

	removingFeeds := []string{}
	for _, dbFeed := range dbFeeds {
		_, ok := loadedFeedMap[dbFeed.Name]
		if !ok {
			log.Info().Str("Player", "Config").Str("Feed", dbFeed.Name).Msg("Feed not found in config")
			removingFeeds = append(removingFeeds, strconv.Itoa(int(*dbFeed.ID)))
		}
	}
	err = db.QueryWithoutResult(ctx, BulkDeleteFeedQuery, map[string]any{"ids": strings.Join(removingFeeds, ",")})
	if err != nil {
		return err
	}

	err = bulkUpsertConfigs(ctx, loadedConfigs)
	if err != nil {
		return err
	}

	whereValues := make([]interface{}, 0, len(loadedConfigs))
	for _, config := range loadedConfigs {
		whereValues = append(whereValues, config.Name)
	}

	configIds, err := db.BulkSelect[ConfigNameIdModel](ctx, "configs", []string{"name", "id"}, []string{"name"}, whereValues)
	if err != nil {
		return err
	}

	configNameIdMap := map[string]int32{}
	for _, configId := range configIds {
		configNameIdMap[configId.Name] = configId.ID
	}

	upsertRows := make([][]any, 0)
	for _, config := range loadedConfigs {
		for _, feed := range config.Feeds {
			configId, ok := configNameIdMap[config.Name]
			if !ok {
				continue
			}
			upsertRows = append(upsertRows, []any{feed.Name, feed.Definition, configId})
		}
	}

	return db.BulkUpsert(ctx, "feeds", []string{"name", "definition", "config_id"}, upsertRows, []string{"name"}, []string{"definition", "config_id"})
}

func Insert(c *fiber.Ctx) error {
	config := new(ConfigInsertModel)
	if err := c.BodyParser(config); err != nil {
		return err
	}

	setDefaultIntervals(config)

	result, err := db.QueryRow[ConfigModel](c.Context(), InsertConfigQuery, map[string]any{
		"name":               config.Name,
		"fetch_interval":     config.FetchInterval,
		"aggregate_interval": config.AggregateInterval,
		"submit_interval":    config.SubmitInterval})
	if err != nil {
		return err
	}

	for _, feed := range config.Feeds {
		feed.ConfigId = &result.ID
		err = db.QueryWithoutResult(c.Context(), InsertFeedQuery, map[string]any{"name": feed.Name, "definition": feed.Definition, "config_id": result.ID})
		if err != nil {
			return err
		}
	}

	return c.JSON(result)
}

func Get(c *fiber.Ctx) error {
	configs, err := db.QueryRows[ConfigModel](c.Context(), SelectConfigQuery, nil)
	if err != nil {
		return err
	}
	return c.JSON(configs)
}

func GetById(c *fiber.Ctx) error {
	id := c.Params("id")
	config, err := db.QueryRow[ConfigModel](c.Context(), SelectConfigByIdQuery, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(config)
}

func DeleteById(c *fiber.Ctx) error {
	id := c.Params("id")
	deleted, err := db.QueryRow[ConfigModel](c.Context(), DeleteConfigQuery, map[string]any{"id": id})
	if err != nil {
		return err
	}
	return c.JSON(deleted)
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
		upsertRows = append(upsertRows, []any{config.Name, config.FetchInterval, config.AggregateInterval, config.SubmitInterval})
	}

	return db.BulkUpsert(ctx, "configs", []string{"name", "fetch_interval", "aggregate_interval", "submit_interval"}, upsertRows, []string{"name"}, []string{"fetch_interval", "aggregate_interval", "submit_interval"})
}

func setDefaultIntervals(config *ConfigInsertModel) {
	if config.FetchInterval == nil || *config.FetchInterval == 0 {
		config.FetchInterval = new(int)
		*config.FetchInterval = 2000
	}
	if config.AggregateInterval == nil || *config.AggregateInterval == 0 {
		config.AggregateInterval = new(int)
		*config.AggregateInterval = 5000
	}
	if config.SubmitInterval == nil || *config.SubmitInterval == 0 {
		config.SubmitInterval = new(int)
		*config.SubmitInterval = 15000
	}
}
