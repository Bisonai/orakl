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

func TestParseInt64MsgParam(t *testing.T) {
	msg := Message{
		Content: MessageContent{
			Args: map[string]any{"testArg": "123"},
		},
	}

	val, err := ParseInt64MsgParam(msg, "testArg")
	if err != nil {
		t.Errorf("Failed to parse int64 param: %v", err)
	}
	if val != 123 {
		t.Errorf("Parsed value did not match expected. Got %v", val)
	}
}

func TestParseStringMsgParam(t *testing.T) {
	msg := Message{
		Content: MessageContent{
			Args: map[string]any{"testArg": "testArg"},
		},
	}

	val, err := ParseStringMsgParam(msg, "testArg")
	if err != nil {
		t.Errorf("Failed to parse string param: %v", err)
	}
	if val != "testArg" {
		t.Errorf("Parsed value did not match expected. Got %v", val)
	}
}
