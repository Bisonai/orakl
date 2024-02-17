package utils

import (
	"log"
	"math/rand"
	"sort"

	"github.com/joho/godotenv"
)

// load env file from root
func LoadEnv() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Print("failed loading .env file, proceeding without .env file")
	}
}

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
