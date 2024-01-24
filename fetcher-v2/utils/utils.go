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

	return fmt.Sprintf("%d", rangeID)
}
