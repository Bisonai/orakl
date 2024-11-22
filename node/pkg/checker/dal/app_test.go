package dal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHandleWsMessage(t *testing.T) {
	// Create a mock context
	ctx := context.Background()

	// Define test cases
	tests := []struct {
		name        string
		inputData   map[string]interface{}
		expected    WsResponse
		expectError bool
	}{
		{
			name: "Valid data",
			inputData: map[string]interface{}{
				"symbol":        "BTC",
				"aggregateTime": "1625151600000",
			},
			expected: WsResponse{
				Symbol:        "BTC",
				AggregateTime: "1625151600000",
			},
			expectError: false,
		},
		{
			name: "Invalid data",
			inputData: map[string]interface{}{
				"symbol":        "BTC",
				"aggregateTime": 1625151600000, // Invalid type
			},
			expected:    WsResponse{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the channel before each test
			for len(wsChan) > 0 {
				<-wsChan
			}

			// Call the function
			err := handleWsMessage(ctx, tt.inputData)

			// Check for errors
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check the channel for the expected data
				select {
				case result := <-wsChan:
					assert.Equal(t, tt.expected, result)
				case <-time.After(1 * time.Second):
					t.Fatal("Expected data not received in channel")
				}
			}
		})
	}
}
