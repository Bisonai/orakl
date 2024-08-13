package keys

import (
	"strconv"
)

func LatestFeedDataKey(feedID int32) string {
	return "latestFeedData:" + strconv.Itoa(int(feedID))
}

func SubmissionDataStreamKey(configId int32) string {
	return "submissionDataStream:" + strconv.Itoa(int(configId))
}

func FeedDataBufferKey() string {
	return "feedDataBuffer"
}
