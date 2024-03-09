package tests

import (
	"testing"

	"bisonai.com/orakl/node/pkg/utils"
)

func TestAvgFloat64OddLength(t *testing.T) {
	// Test with odd length array
	data1 := []float64{1, 2, 3, 4, 5}
	avg1, err := utils.GetFloatAvg(data1)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg1 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg1)
	}
}

func TestAvgFloat64EvnLength(t *testing.T) {
	// Test with even length array
	data2 := []float64{1, 2, 3, 4}
	avg2, err := utils.GetFloatAvg(data2)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg2 != 2.5 {
		t.Errorf("Expected average of 2.5 but got %v", avg2)
	}
}

func TestAvgFloat64Unsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []float64{5, 3, 1, 4, 2}
	avg3, err := utils.GetFloatAvg(data3)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg3 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg3)
	}
}

func TestAvgFloat64ZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []float64{}
	avg4, err := utils.GetFloatAvg(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if avg4 != 0 {
		t.Errorf("Expected average of 0 but got %v", avg4)
	}
}

func TestMedFloat64OddLength(t *testing.T) {
	// Test with odd length array
	data1 := []float64{1, 2, 3, 4, 5}
	med1, err := utils.GetFloatMed(data1)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med1 != 3 {
		t.Errorf("Expected median of 3 but got %v", med1)
	}
}

func TestMedFloat64EvenLength(t *testing.T) {
	// Test with even length array
	data2 := []float64{1, 2, 3, 4}
	med2, err := utils.GetFloatMed(data2)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med2 != 2.5 {
		t.Errorf("Expected median of 2.5 but got %v", med2)
	}
}

func TestMedFloat64Unsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []float64{5, 3, 1, 4, 2}
	med3, err := utils.GetFloatMed(data3)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med3 != 3 {
		t.Errorf("Expected median of 3 but got %v", med3)
	}
}

func TestMedFloat64ZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []float64{}
	med4, err := utils.GetFloatMed(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if med4 != 0 {
		t.Errorf("Expected median of 0 but got %v", med4)
	}
}

func TestAvgIntOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int{1, 2, 3, 4, 5}
	avg1, err := utils.GetIntAvg(data1)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg1 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg1)
	}
}

func TestAvgIntEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int{1, 2, 3, 4}
	avg2, err := utils.GetIntAvg(data2)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg2 != 2 {
		t.Errorf("Expected average of 2 but got %v", avg2)
	}
}

func TestAvgIntZeroLength(t *testing.T) {
	// Test with zero length array
	data3 := []int{}
	avg3, err := utils.GetIntAvg(data3)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if avg3 != 0 {
		t.Errorf("Expected average of 0 but got %v", avg3)
	}
}

func TestAvgIntUnsorted(t *testing.T) {
	// Test with unsorted list
	data4 := []int{5, 3, 1, 4, 2}
	avg4, err := utils.GetIntAvg(data4)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg4 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg4)
	}
}

func TestMedIntOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int{1, 2, 3, 4, 5}
	med1, err := utils.GetMedianInt(data1)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med1 != 3 {
		t.Errorf("Expected median of 3 but got %v", med1)
	}
}

func TestMedIntEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int{1, 2, 3, 4}
	med2, err := utils.GetMedianInt(data2)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med2 != 2 {
		t.Errorf("Expected median of 2 but got %v", med2)
	}
}

func TestMedIntUnsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []int{5, 3, 1, 4, 2}
	med3, err := utils.GetMedianInt(data3)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med3 != 3 {
		t.Errorf("Expected median of 3 but got %v", med3)
	}
}

func TestMedIntZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []int{}
	med4, err := utils.GetMedianInt(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if med4 != 0 {
		t.Errorf("Expected median of 0 but got %v", med4)
	}
}

func TestAvgInt64OddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int64{1, 2, 3, 4, 5}
	avg1, err := utils.GetInt64Avg(data1)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg1 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg1)
	}
}

func TestAvgInt64EvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int64{1, 2, 3, 4}
	avg2, err := utils.GetInt64Avg(data2)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg2 != 2 {
		t.Errorf("Expected average of 2 but got %v", avg2)
	}
}

func TestAvgInt64Unsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []int64{5, 3, 1, 4, 2}
	avg3, err := utils.GetInt64Avg(data3)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg3 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg3)
	}
}

func TestAvgInt64ZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []int64{}
	avg4, err := utils.GetInt64Avg(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if avg4 != 0 {
		t.Errorf("Expected average of 0 but got %v", avg4)
	}
}

func TestMedInt64OddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int64{1, 2, 3, 4, 5}
	med1, err := utils.GetMedianInt64(data1)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med1 != 3 {
		t.Errorf("Expected median of 3 but got %v", med1)
	}
}

func TestMedInt64EvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int64{1, 2, 3, 4}
	med2, err := utils.GetMedianInt64(data2)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med2 != 2 {
		t.Errorf("Expected median of 2 but got %v", med2)
	}
}

func TestMedInt64Unsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []int64{5, 3, 1, 4, 2}
	med3, err := utils.GetMedianInt64(data3)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med3 != 3 {
		t.Errorf("Expected median of 3 but got %v", med3)
	}
}

func TestMedInt64ZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []int64{}
	med4, err := utils.GetMedianInt64(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if med4 != 0 {
		t.Errorf("Expected median of 0 but got %v", med4)
	}
}
