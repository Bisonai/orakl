package common

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"

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
		var def FeedDefinition
		err := json.Unmarshal(feed.Definition, &def)
		if err != nil {
			log.Warn().Err(err).Msg("failed to unmarshal definition")
			continue
		}

		provider := strings.ToLower(def.Provider)
		baseSymbol := strings.ToUpper(def.Base)
		quoteSymbol := strings.ToUpper(def.Quote)

		combinedName := baseSymbol + quoteSymbol
		separatedName := baseSymbol + "-" + quoteSymbol

		if _, exists := feedMaps[provider]; !exists {
			feedMaps[provider] = FeedMaps{
				Combined:  make(map[string][]int32),
				Separated: make(map[string][]int32),
			}
			feedMaps[provider].Combined[combinedName] = []int32{}
			feedMaps[provider].Separated[separatedName] = []int32{}

		}
		feedMaps[provider].Combined[combinedName] = append(feedMaps[provider].Combined[combinedName], feed.ID)
		feedMaps[provider].Separated[separatedName] = append(feedMaps[provider].Separated[separatedName], feed.ID)
	}
	return feedMaps
}

func PriceStringToFloat64(price string) (float64, error) {
	f, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return 0, err
	}

	return FormatFloat64Price(f), nil
}

func VolumeStringToFloat64(volume string) (float64, error) {
	return strconv.ParseFloat(volume, 64)
}

func FormatFloat64Price(price float64) float64 {
	result := price * float64(math.Pow10(DECIMALS))
	if result < 1 {
		result = result * float64(math.Pow10(DECIMALS))
	}
	return result
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

func DecompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
