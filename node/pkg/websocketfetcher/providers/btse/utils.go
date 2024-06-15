package btse

import (
	"time"

	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string]int32, volumeCacheMap *common.VolumeCacheMap) ([]*common.FeedData, error) {
	feedData := make([]*common.FeedData, 0, len(data.Data))

	for _, ticker := range data.Data {
		symbol := ticker.Symbol
		id, ok := feedMap[symbol]
		if !ok {
			log.Warn().Str("Player", "btse").Str("symbol", symbol).Msg("feed not found")
			continue
		}

		timestamp := time.Unix(ticker.Timestamp/1000, 0)
		price := common.FormatFloat64Price(ticker.Price)

		entry := common.FeedData{
			FeedID:    id,
			Value:     price,
			Timestamp: &timestamp,
		}

		volumeData, ok := volumeCacheMap.Map[id]
		if !ok || volumeData.UpdatedAt.Before(time.Now().Add(-common.VolumeCacheLifespan)) {
			entry.Volume = 0
		} else {
			entry.Volume = volumeData.Volume
		}

		feedData = append(feedData, &entry)
	}

	return feedData, nil
}

func FetchVolumes(feedMap map[string]int32, volumeCacheMap *common.VolumeCacheMap) error {
	result, err := request.GetRequest[[]MarketSummary](MARKET_SUMMARY_ENDPOINT, nil, nil)
	if err != nil {
		log.Error().Str("Player", "btse").Err(err).Msg("error in FetchVolumes")
		return err
	}

	for i := range result {
		entry := &result[i]
		symbol := entry.Symbol
		id, exists := feedMap[symbol]
		if !exists {
			continue
		}
		volume := entry.Size

		volumeCacheMap.Mutex.Lock()
		defer volumeCacheMap.Mutex.Unlock()

		volumeCache := volumeCacheMap.Map[id]
		volumeCache.UpdatedAt = time.Now()
		volumeCache.Volume = volume
		volumeCacheMap.Map[id] = volumeCache
	}
	return nil
}
