package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func RandomNumberGenerator() int {
	return rand.Intn(20) + 1
}

func GetIDFromTimestamp(rangeSize int64, t time.Time) string {
	seconds := t.Unix()
	rangeID := seconds / rangeSize
	str := fmt.Sprintf("%d", rangeID)

	if len(str) > 6 {
		str = str[len(str)-6:]
	}

	return str
}

func executeAtEndOfInterval(start time.Time, interval time.Duration, function func()) {
	elapsed := time.Since(start)
	remaining := interval - elapsed

	if remaining > 0 {
		<-time.After(remaining)
		function()
	}
}

func getMaxFromStringSlice(slice []string) (string, error) {
	if len(slice) == 0 {
		return "", errors.New("slice is empty")
	}

	max, err := strconv.Atoi(slice[0])
	if err != nil {
		return "", err
	}

	for _, str := range slice[1:] {
		num, err := strconv.Atoi(str)
		if err != nil {
			return "", err
		}
		if num > max {
			max = num
		}
	}

	return strconv.Itoa(max), nil
}
