package utils

import (
	"errors"
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

func FindMedianInt64(nums []int64) int64 {
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
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

func GetFloatAvg(data []float64) (float64, error) {
	if len(data) == 0 {
		return 0, errors.New("empty array")
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data)), nil
}

func GetFloatMed(data []float64) (float64, error) {
	if len(data) == 0 {
		return 0, errors.New("empty array")
	}
	sort.Float64s(data)
	if len(data)%2 == 0 {
		return (data[len(data)/2-1] + data[len(data)/2]) / 2, nil
	}
	return data[len(data)/2], nil
}
