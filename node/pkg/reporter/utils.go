package reporter

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/chain/helper"
	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/wss"

	"github.com/klaytn/klaytn/common"
	klaytncommon "github.com/klaytn/klaytn/common"
	"github.com/rs/zerolog/log"
)

func GetDeviatingAggregates(submissionPairs map[int32]SubmissionPair, latestData *sync.Map, threshold float64) map[int32]SubmissionPair {
	deviatingSubmissionPairs := make(map[int32]SubmissionPair)
	for configID, submissionPair := range submissionPairs {
		rawLatestData, ok := latestData.Load(submissionPair.Name)
		if !ok {
			log.Warn().Str("Player", "Reporter").Msg("latest data not found during deviation check")
			continue
		}
		latestData, latestDataOk := rawLatestData.(SubmissionData)
		if !latestDataOk {
			log.Error().Str("Player", "Reporter").Msg("latest data type assertion failed during deviation check")
			continue
		}

		shouldReport := ShouldReportDeviation(submissionPair.LastSubmission, latestData.Value, threshold)

		if shouldReport {
			deviatingSubmissionPairs[configID] = submissionPair
		}
	}
	return deviatingSubmissionPairs
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

func ReadOnchainWhitelist(ctx context.Context, chainHelper *helper.ChainHelper, contractAddress string, contractFunction string) ([]common.Address, error) {
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

	arr, ok := rawResultSlice[0].([]common.Address)
	if !ok {
		log.Error().Str("Player", "Reporter").Msg("unexpected raw result type")
		return nil, errorSentinel.ErrReporterResultCastToAddressFail
	}
	return arr, nil
}

func GetDeviationThreshold(submissionInterval time.Duration) float64 {
	if submissionInterval <= 15*time.Second {
		return MIN_DEVIATION_THRESHOLD
	} else if submissionInterval >= 60*time.Minute {
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
		log.Error().Str("Player", "Reporter").Err(jsonMarshalDataErr).Msg("failed to marshal data")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}

	jsonUnmarshalDataErr := json.Unmarshal(jsonMarshalData, &rawSubmissionData)
	if jsonUnmarshalDataErr != nil {
		log.Error().Str("Player", "Reporter").Err(jsonUnmarshalDataErr).Msg("failed to unmarshal data")
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
		log.Error().Str("Player", "Reporter").Err(valueErr).Msg("failed to parse value")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.Value = value

	timestampValue, timestampErr := strconv.ParseInt(rawSubmissionData.AggregateTime, 10, 64)
	if timestampErr != nil {
		log.Error().Str("Player", "Reporter").Err(timestampErr).Msg("failed to parse timestamp")
		return SubmissionData{}, errorSentinel.ErrReporterDalWsDataProcessingFailed
	}
	submissionData.AggregateTime = timestampValue

	return submissionData, nil
}
