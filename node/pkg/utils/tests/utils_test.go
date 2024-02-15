package tests

import (
	"testing"

	"bisonai.com/orakl/node/pkg/utils"
)

func TestAvg(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	avg := utils.GetFloatAvg(data)
	if avg != 3 {
		t.Errorf("Expected 3 but got %v", avg)
	}
}

func TestMed(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	med := utils.GetFloatMed(data)
	if med != 3 {
		t.Errorf("Expected 3 but got %v", med)
	}
}
