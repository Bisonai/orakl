package keys

import (
	"strconv"
)

func SubmissionDataStreamKey(name string) string {
	return "submissionDataStream:" + name
}

func FeedData(feedID int32) string {
	return "feedData:" + strconv.Itoa(int(feedID))
}
