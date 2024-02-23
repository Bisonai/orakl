package tests

import (
	"testing"

	"bisonai.com/orakl/node/pkg/utils"
)

func TestAvgOddLength(t *testing.T) {
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

func TestAvgEvnLength(t *testing.T) {
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

func TestAvgUnsorted(t *testing.T) {
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

func TestAvgZeroLength(t *testing.T) {
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

func TestMedOddLength(t *testing.T) {
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

func TestMedEvenLength(t *testing.T) {
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

func TestMedUnsorted(t *testing.T) {
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

func TestMedZeroLength(t *testing.T) {
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
