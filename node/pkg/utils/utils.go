package utils

import (
	"fmt"
	"math/rand"
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
