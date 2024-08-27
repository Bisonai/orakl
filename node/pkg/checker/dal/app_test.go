package dal

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExtractWsAlarms(t *testing.T) {
	tests := []struct {
		name          string
		messages      []string
		expectedAlert string
	}{
		{
			name:          "Single message",
			messages:      []string{"(BTC) ws delayed by 6(sec)"},
			expectedAlert: "(BTC) ws delayed by 6(sec)",
		},
		{
			name:          "Multiple messages",
			messages:      []string{"(BTC) ws delayed by 6(sec)", "(ETH) ws delayed by 7(sec)"},
			expectedAlert: "(BTC) ws delayed by 6(sec)\n(ETH) ws delayed by 7(sec)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// Set up context with timeout
			for _, msg := range tt.messages {
				wsMsgChan <- msg
			}
			alarmCountMap := map[string]int{
				"BTC": 10,
				"ETH": 10,
			}

			// Call the function
			msgs := extractWsDelayAlarms(ctx, alarmCountMap)

			assert.Equal(t, 0, len(wsMsgChan))

			for i, entry := range tt.messages {
				assert.Equal(t, entry, msgs[i])
			}
		})
	}
}

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

func TestFilterWsReponses(t *testing.T) {
	tests := []struct {
		name          string
		wsResponse    WsResponse
		expectedAlert bool
	}{
		{
			name: "No delay",
			wsResponse: WsResponse{
				Symbol:        "BTC",
				AggregateTime: strconv.FormatInt(time.Now().UnixMilli(), 10),
			},
			expectedAlert: false,
		},
		{
			name: "Delayed response",
			wsResponse: WsResponse{
				Symbol:        "ETH",
				AggregateTime: strconv.FormatInt(time.Now().Add(-10*time.Second).UnixMilli(), 10),
			},
			expectedAlert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset channels
			wsChan = make(chan WsResponse, 1)
			wsMsgChan = make(chan string, 1)

			// Send test data to wsChan
			wsChan <- tt.wsResponse

			// Run filterWsReponses in a goroutine
			go filterDelayedWsResponse()

			// Allow some time for the function to process
			time.Sleep(100 * time.Millisecond)

			// Check if an alert was sent
			select {
			case msg := <-wsMsgChan:
				if !tt.expectedAlert {
					t.Errorf("unexpected alert received: %s", msg)
				}
			default:
				if tt.expectedAlert {
					t.Error("expected alert not received")
				}
			}
		})
	}
}
