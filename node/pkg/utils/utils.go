package utils

import (
	"math/rand"
	"sort"
)

func RandomNumberGenerator() int {
	return rand.Intn(20) + 1
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
