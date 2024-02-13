package bus

// message bus will be passed as parameter to modules that need to communicate with each other

type Message struct {
	From    string
	To      string
	Content string
}

type MessageBus struct {
	channels map[string]chan Message
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		channels: make(map[string]chan Message),
	}
}

func (mb *MessageBus) Subscribe(id string) <-chan Message {
	ch := make(chan Message, 100)
	mb.channels[id] = ch
	return ch
}

func (mb *MessageBus) Publish(msg Message) {
	if ch, ok := mb.channels[msg.To]; ok {
		ch <- msg
	}
}
