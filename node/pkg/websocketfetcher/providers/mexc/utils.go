package mexc

import (
	"time"

	"bisonai.com/miko/node/pkg/websocketfetcher/common"
	"bisonai.com/miko/node/pkg/websocketfetcher/providers/mexc/pb"
)

func WrapperToFeedDataList(wrapper *pb.PushDataV3ApiWrapper, feedMap map[string][]int32) ([]*common.FeedData, error) {
	feedDataList := []*common.FeedData{}

	tickers := wrapper.GetPublicMiniTickers().GetItems()
	if len(tickers) == 0 {
		return feedDataList, nil
	}

	timestamp := time.UnixMilli(wrapper.GetSendTime())

	for _, item := range tickers {
		ids, exists := feedMap[item.GetSymbol()]
		if !exists {
			continue
		}

		value, err := common.PriceStringToFloat64(item.GetPrice())
		if err != nil {
			return feedDataList, err
		}

		// Same inversion the JSON payload had: MEXC's `volume` (field 7) is the
		// rolling turnover, i.e. the quote asset, and `quantity` (field 8) is the
		// traded amount, i.e. the base asset. Checked against
		// GET /api/v3/ticker/24hr for BTCUSDT -- quantity tracks its `volume` and
		// volume tracks its `quoteVolume`. FeedData.Volume is the base asset
		// amount everywhere else, so quantity is the field to take; reading the one
		// actually named `volume` would overstate it by the price of the pair.
		volume, err := common.VolumeStringToFloat64(item.GetQuantity())
		if err != nil {
			return feedDataList, err
		}

		for _, id := range ids {
			feedData := new(common.FeedData)
			feedData.FeedID = id
			feedData.Value = value
			feedData.Timestamp = &timestamp
			feedData.Volume = volume
			feedDataList = append(feedDataList, feedData)
		}
	}

	return feedDataList, nil
}
