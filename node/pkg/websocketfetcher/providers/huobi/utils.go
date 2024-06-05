package huobi

import (
	"fmt"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/websocketfetcher/common"
	"github.com/rs/zerolog/log"
)

func ResponseToFeedData(response Response, feedMap map[string]int32) (*common.FeedData, error) {
	feedData := new(common.FeedData)

	timestamp := time.Unix(response.Ts/1000, 0)
	price := common.FormatFloat64Price(response.Tick.LastPrice)

	splitted := strings.Split(response.Ch, ".")
	if len(splitted) < 3 || splitted[2] != "ticker" {
		log.Error().Str("Ch", response.Ch).Msg("invalid response")
		return feedData, fmt.Errorf("invalid response")
	}

	rawSymbol := splitted[1]
	symbol := strings.ToUpper(rawSymbol)

	id, exists := feedMap[symbol]
	if !exists {
		return feedData, fmt.Errorf("feed not found")
	}
	feedData.FeedID = id
	feedData.Value = price
	feedData.Timestamp = &timestamp
	return feedData, nil
}
