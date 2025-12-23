package bitstamp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func TradeEventToFeedData(data TradeEvent, feedMap map[string][]int32, volumeCacheMap *common.VolumeCacheMap) ([]*common.FeedData, error) {

	rawTimestamp, err := strconv.ParseInt(data.Data.Microtimestamp, 10, 64)
	if err != nil {
		return nil, err
	}

	timestamp := time.Unix(0, rawTimestamp*int64(time.Microsecond))
	value := data.Data.Price
	splitted := strings.Split(data.Channel, "_")
	if len(splitted) < 3 {
		return nil, fmt.Errorf("invalid feed name")
	}
	rawSymbol := splitted[2]
	ids, exists := feedMap[strings.ToUpper(rawSymbol)]
	if !exists {
		return nil, fmt.Errorf("feed not found")
	}

	result := []*common.FeedData{}

	for _, id := range ids {
		feedData := new(common.FeedData)
		feedData.FeedID = id
		feedData.Value = value
		feedData.Timestamp = &timestamp

		volumeData, exists := volumeCacheMap.Get(id)
		if !exists || volumeData.UpdatedAt.Before(time.Now().Add(-common.VolumeCacheLifespan)) {
			feedData.Volume = 0
		} else {
			feedData.Volume = volumeData.Volume
		}
		result = append(result, feedData)
	}

	return result, nil
}

func FetchVolumes(feedMap map[string][]int32, volumeCacheMap *common.VolumeCacheMap) error {
	result, err := request.Request[[]VolumeEntry](request.WithEndpoint(ALL_CURRENCY_PAIR_TICKER_ENDPOINT), request.WithTimeout(common.VolumeFetchTimeout))
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in FetchVolumes")
		return err
	}

	for i := range result {
		entry := &result[i]
		symbol := strings.ReplaceAll(entry.Pair, "/", "")
		ids, exists := feedMap[symbol]
		if !exists {
			continue
		}

		volume, err := common.VolumeStringToFloat64(entry.Volume)
		if err != nil {
			log.Error().Str("Player", "Bitstamp").Err(err).Msg("Failed to convert volume string to float64 in FetchVolumes")
			continue
		}

		for _, id := range ids {
			volumeCacheMap.Set(id, common.VolumeCache{
				UpdatedAt: time.Now(),
				Volume:    volume,
			})
		}
	}

	return nil
}
