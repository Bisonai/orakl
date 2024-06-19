package gemini

import (
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func TradeResponseToFeedDataList(data Response, feedMap map[string]int32, volumeCacheMap *common.VolumeCacheMap) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	timestamp := time.Unix(*data.TimestampMs/1000, 0)
	for _, event := range data.Events {
		feedData := new(common.FeedData)
		id, exists := feedMap[event.Symbol]
		if !exists {
			log.Warn().Str("Player", "Gemini").Str("key", event.Symbol).Msg("feed not found")
			continue
		}

		price, err := common.PriceStringToFloat64(event.Price)
		if err != nil {
			log.Warn().Str("Player", "Gemini").Err(err).Msg("error in PriceStringToFloat64")
			continue
		}
		feedData.FeedID = id
		feedData.Value = price
		feedData.Timestamp = &timestamp
		volumeData, exists := volumeCacheMap.Map[id]
		if !exists || volumeData.UpdatedAt.Before(time.Now().Add(-common.VolumeCacheLifespan)) {
			feedData.Volume = 0
		} else {
			feedData.Volume = volumeData.Volume
		}

		feedDataList = append(feedDataList, feedData)
	}

	return feedDataList, nil
}

func FetchVolumes(feedMap map[string]int32, volumeCacheMap *common.VolumeCacheMap) {
	for symbol, id := range feedMap {
		endpoint := TICKER_ENDPOINT + strings.ToLower(symbol)
		result, err := request.GetRequest[HttpTickerResponse](endpoint, nil, nil)
		if err != nil {
			log.Error().Str("Player", "Gemini").Err(err).Msg("fetch volumes, http request failed")
			continue
		}
		timestampRaw, ok := result.Volume["timestamp"].(float64)
		if !ok {
			log.Error().Str("Player", "Gemini").Msg("fetch volumes, entry timestamp not found")
			continue
		}

		timestamp := time.Unix(int64(timestampRaw)/1000, 0)

		for key, value := range result.Volume {
			if strings.HasPrefix(symbol, key) {
				volumeStr, ok := value.(string)
				if !ok {
					log.Error().Str("Player", "Gemini").Msg("error in parsing volume to string")
					continue
				}
				volume, err := common.VolumeStringToFloat64(volumeStr)
				if err != nil {
					log.Error().Str("Player", "Gemini").Err(err).Msg("error in VolumeStringToFloat64")
					continue
				}

				volumeCacheMap.Mutex.Lock()
				volumeCacheMap.Map[id] = common.VolumeCache{
					UpdatedAt: timestamp,
					Volume:    volume,
				}
				volumeCacheMap.Mutex.Unlock()
			}
		}

		// gemini recommends 1 request per second
		time.Sleep(1 * time.Second)
	}
}
