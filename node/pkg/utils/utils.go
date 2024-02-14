package utils

import (
	"fmt"
	"math/rand"
	"sort"
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

func FindMedian(nums []int) int {
	sort.Ints(nums)
	n := len(nums)
	if n%2 == 0 {
		// Round down
		return (nums[n/2-1] + nums[n/2]) / 2

		// Or round to nearest integer
		// return int(float64(nums[n/2-1]+nums[n/2]) / 2 + 0.5)
	} else {
		return nums[n/2]
	}
}
