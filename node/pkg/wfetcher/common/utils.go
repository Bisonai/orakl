package common

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	"github.com/rs/zerolog/log"
)

/*
Generates two types of maps with different keys for the same feed ID.
The combined map has keys like "BTCUSD", and the separated map has keys like "BTC-USD".
The reason for this is that different exchanges use different naming conventions.
*/
func GetWssFeedMap(feeds []Feed) map[string]FeedMaps {
	feedMaps := make(map[string]FeedMaps)
	for _, feed := range feeds {
		if !strings.Contains(feed.Name, "wss") {
			continue
		}

		raw := strings.Split(feed.Name, "-")
		if len(raw) != 4 {
			log.Warn().Str("name", feed.Name).Msg("invalid name")
			continue
		}

		provider := strings.ToLower(raw[0])
		base := strings.ToUpper(raw[2])
		quote := strings.ToUpper(raw[3])
		combinedName := base + quote
		separatedName := base + "-" + quote

		if _, exists := feedMaps[provider]; !exists {
			feedMaps[provider] = FeedMaps{
				Combined:  make(map[string]int32),
				Separated: make(map[string]int32),
			}
		}
		feedMaps[provider].Combined[combinedName] = feed.ID
		feedMaps[provider].Separated[separatedName] = feed.ID
	}
	return feedMaps
}

func StoreFeed(ctx context.Context, feedData FeedData, expiration time.Duration) error {
	err := db.SetObject(ctx, "latestFeedData:"+strconv.Itoa(int(feedData.FeedId)), feedData, expiration)
	if err != nil {
		return err
	}
	return db.LPushObject(ctx, "feedDataBuffer", []FeedData{feedData})
}

func StoreFeeds(ctx context.Context, feedData []FeedData) error {
	latestData := make(map[string]any)
	for _, data := range feedData {
		key := "latestFeedData:" + strconv.Itoa(int(data.FeedId))
		if latestData[key] != nil && latestData[key].(FeedData).Timestamp.After(*data.Timestamp) {
			continue
		}
		latestData[key] = data
	}
	err := db.MSetObject(ctx, latestData)
	if err != nil {
		return err
	}
	return db.LPushObject(ctx, "feedDataBuffer", feedData)
}

func PriceStringToFloat64(price string) (float64, error) {
	f, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0, err
	}

	return f * float64(math.Pow10(int(DECIMALS))), nil
}

func MessageToStruct[T any](message map[string]any) (T, error) {
	var result T
	data, err := json.Marshal(message)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
