package tests

import (
	"testing"

	"bisonai.com/miko/node/pkg/utils/calculator"
)

func TestFloatAvgOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []float64{1, 2, 3, 4, 5}
	avg1, err := calculator.GetFloatAvg(data1)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg1 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg1)
	}
}

func TestFloatAvgEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []float64{1, 2, 3, 4}
	avg2, err := calculator.GetFloatAvg(data2)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg2 != 2.5 {
		t.Errorf("Expected average of 2.5 but got %v", avg2)
	}
}

func TestFloatAvgUnsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []float64{5, 3, 1, 4, 2}
	avg3, err := calculator.GetFloatAvg(data3)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg3 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg3)
	}
}

func TestFloatAvgZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []float64{}
	avg4, err := calculator.GetFloatAvg(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if avg4 != 0 {
		t.Errorf("Expected average of 0 but got %v", avg4)
	}
}

func TestFloatMedOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []float64{1, 2, 3, 4, 5}
	med1, err := calculator.GetFloatMed(data1)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med1 != 3 {
		t.Errorf("Expected median of 3 but got %v", med1)
	}
}

func TestFloatMedEvenLength(t *testing.T) {
	// Test with even length array
	data2 := []float64{1, 2, 3, 4}
	med2, err := calculator.GetFloatMed(data2)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med2 != 2.5 {
		t.Errorf("Expected median of 2.5 but got %v", med2)
	}
}

func TestFloatMedUnsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []float64{5, 3, 1, 4, 2}
	med3, err := calculator.GetFloatMed(data3)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med3 != 3 {
		t.Errorf("Expected median of 3 but got %v", med3)
	}
}

func TestFloatMedZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []float64{}
	med4, err := calculator.GetFloatMed(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if med4 != 0 {
		t.Errorf("Expected median of 0 but got %v", med4)
	}
}

func TestIntAvgOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int{1, 2, 3, 4, 5}
	avg1, err := calculator.GetIntAvg(data1)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg1 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg1)
	}
}

func TestIntAvgEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int{1, 2, 3, 4}
	avg2, err := calculator.GetIntAvg(data2)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg2 != 2 {
		t.Errorf("Expected average of 2 but got %v", avg2)
	}
}

func TestIntAvgZeroLength(t *testing.T) {
	// Test with zero length array
	data3 := []int{}
	avg3, err := calculator.GetIntAvg(data3)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if avg3 != 0 {
		t.Errorf("Expected average of 0 but got %v", avg3)
	}
}

func TestIntAvgUnsorted(t *testing.T) {
	// Test with unsorted list
	data4 := []int{5, 3, 1, 4, 2}
	avg4, err := calculator.GetIntAvg(data4)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg4 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg4)
	}
}

func TestIntMedOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int{1, 2, 3, 4, 5}
	med1, err := calculator.GetIntMed(data1)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med1 != 3 {
		t.Errorf("Expected median of 3 but got %v", med1)
	}
}

func TestIntMedEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int{1, 2, 3, 4}
	med2, err := calculator.GetIntMed(data2)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med2 != 2 {
		t.Errorf("Expected median of 2 but got %v", med2)
	}
}

func TestIntMedUnsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []int{5, 3, 1, 4, 2}
	med3, err := calculator.GetIntMed(data3)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med3 != 3 {
		t.Errorf("Expected median of 3 but got %v", med3)
	}
}

func TestIntMedZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []int{}
	med4, err := calculator.GetIntMed(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if med4 != 0 {
		t.Errorf("Expected median of 0 but got %v", med4)
	}
}

func TestInt64AvgOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int64{1, 2, 3, 4, 5}
	avg1, err := calculator.GetInt64Avg(data1)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg1 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg1)
	}
}

func TestInt64AvgEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int64{1, 2, 3, 4}
	avg2, err := calculator.GetInt64Avg(data2)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg2 != 2 {
		t.Errorf("Expected average of 2 but got %v", avg2)
	}
}

func TestInt64AvgUnsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []int64{5, 3, 1, 4, 2}
	avg3, err := calculator.GetInt64Avg(data3)
	if err != nil {
		t.Errorf("Error calculating average: %v", err)
	}
	if avg3 != 3 {
		t.Errorf("Expected average of 3 but got %v", avg3)
	}
}

func TestInt64AvgZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []int64{}
	avg4, err := calculator.GetInt64Avg(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if avg4 != 0 {
		t.Errorf("Expected average of 0 but got %v", avg4)
	}
}

func TestInt64MedOddLength(t *testing.T) {
	// Test with odd length array
	data1 := []int64{1, 2, 3, 4, 5}
	med1, err := calculator.GetInt64Med(data1)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med1 != 3 {
		t.Errorf("Expected median of 3 but got %v", med1)
	}
}

func TestInt64MedEvnLength(t *testing.T) {
	// Test with even length array
	data2 := []int64{1, 2, 3, 4}
	med2, err := calculator.GetInt64Med(data2)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med2 != 2 {
		t.Errorf("Expected median of 2 but got %v", med2)
	}
}

func TestInt64MedUnsorted(t *testing.T) {
	// Test with unsorted list
	data3 := []int64{5, 3, 1, 4, 2}
	med3, err := calculator.GetInt64Med(data3)
	if err != nil {
		t.Errorf("Error calculating median: %v", err)
	}
	if med3 != 3 {
		t.Errorf("Expected median of 3 but got %v", med3)
	}
}

func TestInt64MedZeroLength(t *testing.T) {
	// Test with zero length array
	data4 := []int64{}
	med4, err := calculator.GetInt64Med(data4)
	if err == nil {
		t.Errorf("Expected error but got nil")
	}
	if med4 != 0 {
		t.Errorf("Expected median of 0 but got %v", med4)
	}
}
