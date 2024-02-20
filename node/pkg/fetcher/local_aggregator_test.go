package fetcher

import (
	"testing"
)

func TestAvg(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	avg := getAvg(data)
	if avg != 3 {
		t.Errorf("Expected 3 but got %v", avg)
	}
}

func TestMed(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	med := getMed(data)
	if med != 3 {
		t.Errorf("Expected 3 but got %v", med)
	}
}
