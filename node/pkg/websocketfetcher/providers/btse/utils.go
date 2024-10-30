package btse

import (
	"time"

	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedDataList(data Response, feedMap map[string][]int32, volumeCacheMap *common.VolumeCacheMap) ([]*common.FeedData, error) {
	feedData := make([]*common.FeedData, 0, len(data.Data))

	for _, ticker := range data.Data {
		symbol := ticker.Symbol
		ids, ok := feedMap[symbol]
		if !ok {
			log.Warn().Str("Player", "btse").Str("symbol", symbol).Msg("feed not found")
			continue
		}

		timestamp := time.UnixMilli(ticker.Timestamp)
		price := common.FormatFloat64Price(ticker.Price)

		for _, id := range ids {
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
	}

	return feedData, nil
}

func FetchVolumes(feedMap map[string][]int32, volumeCacheMap *common.VolumeCacheMap) error {
	result, err := request.Request[[]MarketSummary](request.WithEndpoint(MARKET_SUMMARY_ENDPOINT), request.WithTimeout(common.VolumeFetchTimeout))
	if err != nil {
		log.Error().Str("Player", "btse").Err(err).Msg("error in FetchVolumes")
		return err
	}

	for i := range result {
		entry := &result[i]
		symbol := entry.Symbol
		ids, exists := feedMap[symbol]
		if !exists {
			continue
		}
		volume := entry.Size

		for _, id := range ids {
			volumeCacheMap.Mutex.Lock()
			volumeCacheMap.Map[id] = common.VolumeCache{
				UpdatedAt: time.Now(),
				Volume:    volume,
			}
			volumeCacheMap.Mutex.Unlock()
		}
	}
	return nil
}
