package fetcher

import (
	"sort"
)

func getAvg(data []float64) float64 {
	var sum float64
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func getMed(data []float64) float64 {
	sort.Float64s(data)
	if len(data)%2 == 0 {
		return (data[len(data)/2-1] + data[len(data)/2]) / 2
	}
	return data[len(data)/2]
}
