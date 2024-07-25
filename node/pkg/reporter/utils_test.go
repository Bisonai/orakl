//nolint:all

package reporter

import (
	"reflect"
	"testing"
)

func TestProcessDalWsRawData(t *testing.T) {
	input := RawSubmissionData{
		Value:         "123",
		AggregateTime: "1609459200", // 2021-01-01 00:00:00 UTC
		Proof:         "0xabcdef",
		FeedHash:      "0x123456",
	}
	expected := SubmissionData{
		Value:         123,
		AggregateTime: 1609459200,
		Proof:         []byte{0xab, 0xcd, 0xef},
		FeedHash:      [32]byte{0x12, 0x34, 0x56},
	}

	result, err := ProcessDalWsRawData(input)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %+v, got %+v", expected, result)
	}
}
