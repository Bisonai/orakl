package keys

import (
	"strconv"
)

func LatestFeedDataKey(feedID int32) string {
	return "latestFeedData:" + strconv.Itoa(int(feedID))
}

func LocalAggregateKey(configID int32) string {
	return "localAggregate:" + strconv.Itoa(int(configID))
}

func GlobalAggregateKey(configID int32) string {
	return "globalAggregate:" + strconv.Itoa(int(configID))
}

func ProofKey(configID int32, round int32) string {
	return "proof:" + strconv.Itoa(int(configID)) + "|round:" + strconv.Itoa(int(round))
}

func LastSubmissionKey(configID int32) string {
	return "lastSubmission:" + strconv.Itoa(int(configID))
}

func FeedDataBufferKey() string {
	return "feedDataBuffer"
}
