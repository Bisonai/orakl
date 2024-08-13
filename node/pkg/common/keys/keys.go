package keys

import (
	"strconv"
)

func SubmissionDataStreamKey(configId int32) string {
	return "submissionDataStream:" + strconv.Itoa(int(configId))
}
