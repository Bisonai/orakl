package bus

import (
	"testing"
)

func TestSubscribeAndPublish(t *testing.T) {
	bus := NewMessageBus()

	// Test Subscribe
	channel := bus.Subscribe("test", 10)

	// Test Publish
	bus.Publish(Message{
		From:    "testFrom",
		To:      "test",
		Content: "testContent",
	})

	select {
	case msg := <-channel:
		if msg.From != "testFrom" || msg.To != "test" || msg.Content.(string) != "testContent" {
			t.Errorf("Message did not match expected. Got %v", msg)
		}
	default:
		t.Errorf("No message received on channel")
	}
}
