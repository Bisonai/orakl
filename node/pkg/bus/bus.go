package bus

import (
	"errors"
	"strconv"
)

// message bus will be passed as parameter to modules that need to communicate with each other

type Message struct {
	From    string
	To      string
	Content MessageContent
}

type MessageContent struct {
	Command string
	Args    map[string]any
}

type MessageBus struct {
	channels  map[string]chan Message
	msgBuffer int
}

func New(bufferSize int) *MessageBus {
	return &MessageBus{
		channels:  make(map[string]chan Message),
		msgBuffer: bufferSize,
	}
}

func (mb *MessageBus) Subscribe(id string) <-chan Message {
	ch := make(chan Message, mb.msgBuffer)
	mb.channels[id] = ch
	return ch
}

func (mb *MessageBus) Publish(msg Message) error {
	ch, ok := mb.channels[msg.To]
	if !ok {
		return errors.New("channel not found")
	}
	select {
	case ch <- msg:
		return nil
	default:
		return errors.New("failed to send message to channel")
	}
}

func ParseInt64MsgParam(msg Message, param string) (int64, error) {
	rawId, ok := msg.Content.Args[param]
	if !ok {
		return 0, errors.New("param not found in message")
	}

	idPayload, ok := rawId.(string)
	if !ok {
		return 0, errors.New("failed to convert adapter id to string")
	}

	id, err := strconv.ParseInt(idPayload, 10, 64)
	if err != nil {
		return 0, errors.New("failed to parse adapterId")
	}

	return id, nil
}

func ParseStringMsgParam(msg Message, param string) (string, error) {
	raw, ok := msg.Content.Args[param]
	if !ok {
		return "", errors.New("param not found in message")
	}

	payload, ok := raw.(string)
	if !ok {
		return "", errors.New("failed to convert param to string")
	}

	return payload, nil
}
