package bus

import (
	"testing"
)

func TestSubscribeAndPublish(t *testing.T) {
	mb := NewMessageBus()

	// Test Subscribe
	channel := mb.Subscribe("test", 10)

	// Test Publish
	mb.Publish(Message{
		From: "testFrom",
		To:   "test",
		Content: MessageContent{
			Command: "testCommand",
			Args:    map[string]any{"testArg": "testArg"},
		},
	})

	select {
	case msg := <-channel:
		if msg.From != "testFrom" || msg.To != "test" || msg.Content.Command != "testCommand" {
			t.Errorf("Message did not match expected. Got %v", msg)
		}
	default:
		t.Errorf("No message received on channel")
	}
}
