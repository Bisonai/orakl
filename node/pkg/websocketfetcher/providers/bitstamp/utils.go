package bitstamp

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/utils/request"
	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func TradeEventToFeedData(data TradeEvent, feedMap map[string]int32, volumeCacheMap *common.VolumeCacheMap) (*common.FeedData, error) {
	feedData := new(common.FeedData)
	rawTimestamp, err := strconv.ParseInt(data.Data.Microtimestamp, 10, 64)
	if err != nil {
		return feedData, err
	}

	timestamp := time.Unix(0, rawTimestamp*int64(time.Microsecond))
	value := common.FormatFloat64Price(data.Data.Price)
	splitted := strings.Split(data.Channel, "_")
	if len(splitted) < 3 {
		return feedData, fmt.Errorf("invalid feed name")
	}
	rawSymbol := splitted[2]
	id, exists := feedMap[strings.ToUpper(rawSymbol)]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = value
	feedData.Timestamp = &timestamp

	volumeData, exists := volumeCacheMap.Map[id]
	if !exists || volumeData.UpdatedAt.Before(time.Now().Add(-common.VolumeCacheLifespan)) {
		feedData.Volume = 0
	} else {
		feedData.Volume = volumeData.Volume
	}

	return feedData, nil
}

func FetchVolumes(feedMap map[string]int32, volumeCacheMap *common.VolumeCacheMap) error {
	result, err := request.GetRequest[[]VolumeEntry](ALL_CURRENCY_PAIR_TICKER_ENDPOINT, nil, nil)
	if err != nil {
		log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in FetchVolumes")
		return err
	}

	for i := range result {
		entry := &result[i]
		symbol := strings.ReplaceAll(entry.Pair, "/", "")
		id, exists := feedMap[symbol]
		if !exists {
			continue
		}

		volume, err := common.VolumeStringToFloat64(entry.Volume)
		if err != nil {
			fmt.Println(err)
			log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in VolumeStringToFloat64")
			continue
		}

		// timestamp of volume data too quite old that it is older than 10 minutes
		// since the data kept not being utilized, use time.Now() instead of api call time
		// even though it isn't actually updated now, the value holds latest snapshot.
		// if this behavior is not desired, use following commented code

		// NumTimestamp, err := strconv.ParseInt(entry.Timestamp, 10, 64)
		// if err != nil {
		// 	log.Error().Str("Player", "Bitstamp").Err(err).Msg("error in ParseInt")
		// 	continue
		// }
		// time := time.Unix(NumTimestamp, 0)

		volumeCacheMap.Mutex.Lock()
		volumeCacheMap.Map[id] = common.VolumeCache{
			UpdatedAt: time.Now(),
			Volume:    volume,
		}
		volumeCacheMap.Mutex.Unlock()
	}

	return nil
}
