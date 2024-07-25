//nolint:all

package reporter

import (
	"reflect"
	"testing"
)

func TestProcessDalWsRawData(t *testing.T) {
	tests := []struct {
		name     string
		input    RawSubmissionData
		expected SubmissionData
		wantErr  bool
	}{
		{
			name: "Valid input",
			input: RawSubmissionData{
				Value:         "123",
				AggregateTime: "1609459200",
				Proof:         "0xabcdef",
				FeedHash:      "0x123456",
			},
			expected: SubmissionData{
				Value:         123,
				AggregateTime: 1609459200,
				Proof:         []byte{0xab, 0xcd, 0xef},
				FeedHash:      [32]byte{0x12, 0x34, 0x56},
			},
			wantErr: false,
		},
		{
			name: "Invalid value",
			input: RawSubmissionData{
				Value:         "abc",
				AggregateTime: "1609459200",
				Proof:         "0xabcdef",
				FeedHash:      "0x123456",
			},
			expected: SubmissionData{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessDalWsRawData(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessDalWsRawData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ProcessDalWsRawData() = %v, want %v", result, tt.expected)
			}
		})
	}
}
