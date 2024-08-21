package calculator

import (
	"math/rand"
	"sort"

	errorSentinel "bisonai.com/miko/node/pkg/error"
)

func RandomNumberGenerator() int {
	return rand.Intn(20) + 1
}

func GetIntAvg(nums []int) (int, error) {
	if len(nums) == 0 {
		return 0, errorSentinel.ErrCalculatorEmptyArr
	}
	var sum int
	for _, v := range nums {
		sum += v
	}
	return sum / len(nums), nil
}

func GetIntMed(nums []int) (int, error) {
	if len(nums) == 0 {
		return 0, errorSentinel.ErrCalculatorEmptyArr
	}
	sort.Ints(nums)
	n := len(nums)
	if n%2 == 0 {
		// Round down
		return (nums[n/2-1] + nums[n/2]) / 2, nil
	} else {
		return nums[n/2], nil
	}
}

func GetInt64Avg(nums []int64) (int64, error) {
	if len(nums) == 0 {
		return 0, errorSentinel.ErrCalculatorEmptyArr
	}
	var sum int64
	for _, v := range nums {
		sum += v
	}
	return sum / int64(len(nums)), nil
}

func GetInt64Med(nums []int64) (int64, error) {
	if len(nums) == 0 {
		return 0, errorSentinel.ErrCalculatorEmptyArr
	}

	if len(nums) == 1 {
		return nums[0], nil
	}

	if len(nums) == 2 {
		return (nums[0] + nums[1]) / 2, nil
	}

	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	n := len(nums)
	if n%2 == 0 {
		// Round down
		return (nums[n/2-1] + nums[n/2]) / 2, nil
	} else {
		return nums[n/2], nil
	}
}

func GetFloatAvg(data []float64) (float64, error) {
	if len(data) == 0 {
		return 0, errorSentinel.ErrCalculatorEmptyArr
	}
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data)), nil
}

func GetFloatMed(data []float64) (float64, error) {
	if len(data) == 0 {
		return 0, errorSentinel.ErrCalculatorEmptyArr
	}
	sort.Float64s(data)
	if len(data)%2 == 0 {
		return (data[len(data)/2-1] + data[len(data)/2]) / 2, nil
	}
	return data[len(data)/2], nil
}
