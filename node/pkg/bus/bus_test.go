package bus

import (
	"errors"
	"testing"
	"time"
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

func TestMessageResponse(t *testing.T) {
	// Create a new message bus
	mb := New(10)

	// Subscribe to a channel
	ch := mb.Subscribe("test")

	// Create a message with a response channel
	msg := Message{
		From:     "sender",
		To:       "test",
		Content:  MessageContent{Command: "do something", Args: nil},
		Response: make(chan MessageResponse),
	}

	// Send the message
	err := mb.Publish(msg)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// In a separate goroutine, receive the message, process it, and send a response
	go func() {
		select {
		case receivedMsg := <-ch:
			// Check the received message
			if receivedMsg.From != "sender" || receivedMsg.To != "test" || receivedMsg.Content.Command != "do something" {
				t.Errorf("Received message did not match expected. Got %v", receivedMsg)
			}

			// Send a response
			receivedMsg.Response <- MessageResponse{Success: true, Args: nil}
		case <-time.After(5 * time.Second):
			t.Errorf("No message received on channel")
		}
	}()

	// Receive the response and check it
	select {
	case response := <-msg.Response:
		if !response.Success {
			t.Errorf("Received response did not indicate success. Got %v", response)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("No response received on response channel")
	}
}

func TestHandleMessageError(t *testing.T) {
	// Create a new message bus
	mb := New(10)

	// Subscribe to a channel
	ch := mb.Subscribe("test")

	// Create a message with a response channel
	msg := Message{
		From:     "sender",
		To:       "test",
		Content:  MessageContent{Command: "do something", Args: nil},
		Response: make(chan MessageResponse),
	}

	// Send the message
	err := mb.Publish(msg)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// In a separate goroutine, receive the message, process it, and send a response
	go func() {
		select {
		case receivedMsg := <-ch:
			// Check the received message
			if receivedMsg.From != "sender" || receivedMsg.To != "test" || receivedMsg.Content.Command != "do something" {
				t.Errorf("Received message did not match expected. Got %v", receivedMsg)
			}

			// Send an error response
			HandleMessageError(errors.New("test error"), receivedMsg, "test error message")
		case <-time.After(5 * time.Second):
			t.Errorf("No message received on channel")
		}
	}()

	// Receive the response and check it
	select {
	case response := <-msg.Response:
		if response.Success {
			t.Errorf("Received response indicated success. Got %v", response)
		} else {
			if response.Args["error"] != "test error" {
				t.Errorf("Received response did not contain expected error. Got %v", response)
			}
		}
	case <-time.After(5 * time.Second):
		t.Errorf("No response received on response channel")
	}
}
