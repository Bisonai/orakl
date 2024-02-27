package bus

import (
	"testing"
)

func TestSubscribeAndPublish(t *testing.T) {
	mb := New(10)

	// Test Subscribe
	channel := mb.Subscribe("test")

	// Test Publish
	err := mb.Publish(Message{
		From: "testFrom",
		To:   "test",
		Content: MessageContent{
			Command: "testCommand",
			Args:    map[string]any{"testArg": "testArg"},
		},
	})
	if err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	select {
	case msg := <-channel:
		if msg.From != "testFrom" || msg.To != "test" || msg.Content.Command != "testCommand" {
			t.Errorf("Message did not match expected. Got %v", msg)
		}
	default:
		t.Errorf("No message received on channel")
	}
}
