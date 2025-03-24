package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/miko/node/pkg/chain/helper"
	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/secrets"
	"bisonai.com/miko/node/pkg/utils/request"
	"bisonai.com/miko/node/pkg/wss"

	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func GetDeviatingAggregates(latestSubmittedData *sync.Map, latestData *sync.Map, threshold float64) map[string]SubmissionData {
	deviatingSubmissionPairs := map[string]SubmissionData{}
	latestSubmittedData.Range(func(key, value any) bool {
		pair := key.(string)
		oldValue := value.(int64)
		newValue, ok := GetLatestData(latestData, pair)

		if !ok {
			log.Error().Str("Player", "Reporter").Msg("latest data not found during deviation check")
			return true
		}

		if ShouldReportDeviation(oldValue, newValue.Value, threshold) {
			deviatingSubmissionPairs[pair] = newValue
		}
		return true
	})

	return deviatingSubmissionPairs
}

func GetLatestDataRest(ctx context.Context, baseEndpoint string, name []string) (map[string]SubmissionData, error) {
	url := fmt.Sprintf("%s/latest-data-feeds-unstrict/%s", baseEndpoint, strings.Join(name, ","))
	resp, err := request.Request[[]RawSubmissionData](
		request.WithEndpoint(url),
		request.WithHeaders(map[string]string{"X-API-Key": secrets.GetSecret("API_KEY")}),
		request.WithTimeout(5*time.Second), // data will be too old if it takes too long to fetch
	)
	if err != nil {
		return nil, err
	}

	result := map[string]SubmissionData{}

	for _, entry := range resp {
		submissionData, err := RawSubmissionData2SubmissionData(entry)
		if err != nil {
			return nil, err
		}
		result[entry.Symbol] = submissionData
	}

	return result, nil
}

func GetLatestData(latestDataMap *sync.Map, name string) (SubmissionData, bool) {
	rawLatestData, ok := latestDataMap.Load(name)
	if !ok {
		log.Debug().Str("Player", "Reporter").Msg("latest data not found during deviation check")
		return SubmissionData{}, false
	}
	convertedLatestData, latestDataOk := rawLatestData.(SubmissionData)
	if !latestDataOk {
		log.Error().Str("Player", "Reporter").Msg("latest data type assertion failed during deviation check")
		return SubmissionData{}, false
	}
	return convertedLatestData, true
}

func ShouldReportDeviation(oldValue int64, newValue int64, threshold float64) bool {
	denominator := math.Pow10(DECIMALS)
	oldValueInFLoat := float64(oldValue) / denominator
	newValueInFLoat := float64(newValue) / denominator

	if oldValue != 0 && newValue != 0 {
		deviationRange := oldValueInFLoat * threshold
		minimum := oldValueInFLoat - deviationRange
		maximum := oldValueInFLoat + deviationRange
		return newValueInFLoat < minimum || newValueInFLoat > maximum
	} else if oldValue == 0 && newValue != 0 {
		return newValueInFLoat > DEVIATION_ABSOLUTE_THRESHOLD
	} else {
		return false
	}
}

func ReadOnchainWhitelist(ctx context.Context, chainHelper *helper.ChainHelper, contractAddress string, contractFunction string) ([]klaytncommon.Address, error) {
	result, err := chainHelper.ReadContract(ctx, contractAddress, contractFunction)
	if err != nil {
		log.Error().Str("Player", "Reporter").Err(err).Msg("failed to read contract")
		return nil, err
	}

	rawResultSlice, ok := result.([]interface{})
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errorSentinel.ErrReporterResultCastToInterfaceFail
	}

	arr, ok := rawResultSlice[0].([]klaytncommon.Address)
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errorSentinel.ErrReporterResultCastToAddressFail
	}
	return arr, nil
}

func GetDeviationThreshold(submissionInterval time.Duration) float64 {
	if submissionInterval <= 15*time.Second {
		return MIN_DEVIATION_THRESHOLD
	} else if submissionInterval >= 60*time.Second {
		return MAX_DEVIATION_THRESHOLD
	} else {
		submissionIntervalSec := submissionInterval.Seconds()
		return MIN_DEVIATION_THRESHOLD - ((submissionIntervalSec-MIN_INTERVAL)/(MAX_INTERVAL-MIN_INTERVAL))*(MIN_DEVIATION_THRESHOLD-MAX_DEVIATION_THRESHOLD)
	}
}

func SetupDalWsHelper(ctx context.Context, configs []Config, endpoint string, apiKey string) (*wss.WebsocketHelper, error) {
	subscription := Subscription{
		Method: "SUBSCRIBE",
		Params: []string{},
	}

	for _, configs := range configs {
		subscription.Params = append(subscription.Params, "submission@"+configs.Name)
	}

	wsHelper, wsHelperErr := wss.NewWebsocketHelper(
		ctx,
		wss.WithEndpoint(endpoint),
		wss.WithSubscriptions([]interface{}{subscription}),
		wss.WithRequestHeaders(map[string]string{"X-API-Key": apiKey}),
	)
	if wsHelperErr != nil {
		log.Error().Str("Player", "Reporter").Err(wsHelperErr).Msg("failed to create websocket helper")
		return nil, wsHelperErr
	}
	return wsHelper, nil

}

func ProcessDalWsRawData(data any) (SubmissionData, error) {
	rawSubmissionData := RawSubmissionData{}

	jsonMarshalData, jsonMarshalDataErr := json.Marshal(data)
	if jsonMarshalDataErr != nil {
		log.Error().Str("Player", "Reporter").Err(jsonMarshalDataErr).Msg("failed to marshal data: " + fmt.Sprintf("%v", data))
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}

	jsonUnmarshalDataErr := json.Unmarshal(jsonMarshalData, &rawSubmissionData)
	if jsonUnmarshalDataErr != nil {
		log.Error().Str("Player", "Reporter").Err(jsonUnmarshalDataErr).Msg("failed to unmarshal data: " + string(jsonMarshalData))
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}

	return RawSubmissionData2SubmissionData(rawSubmissionData)
}

func RawSubmissionData2SubmissionData(rawSubmissionData RawSubmissionData) (SubmissionData, error) {
	if rawSubmissionData.FeedHash == "" || rawSubmissionData.Proof == "" || rawSubmissionData.Value == "" || rawSubmissionData.AggregateTime == "" {
		log.Error().Str("Player", "Reporter").Msg("empty data fields")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	feedHashBytes := klaytncommon.Hex2Bytes(strings.TrimPrefix(rawSubmissionData.FeedHash, "0x"))
	feedHash := [32]byte{}
	copy(feedHash[:], feedHashBytes)
	submissionData := SubmissionData{
		FeedHash: feedHash,
		Proof:    klaytncommon.Hex2Bytes(strings.TrimPrefix(rawSubmissionData.Proof, "0x")),
	}

	value, valueErr := strconv.ParseInt(rawSubmissionData.Value, 10, 64)
	if valueErr != nil {
		log.Error().Str("Player", "Reporter").Err(valueErr).Msg("failed to parse value: " + rawSubmissionData.Value)
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.Value = value

	timestampValue, timestampErr := strconv.ParseInt(rawSubmissionData.AggregateTime, 10, 64)
	if timestampErr != nil {
		log.Error().Str("Player", "Reporter").Err(timestampErr).Msg("failed to parse timestamp: " + rawSubmissionData.AggregateTime)
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.AggregateTime = timestampValue
	submissionData.Symbol = rawSubmissionData.Symbol

	return submissionData, nil
}
