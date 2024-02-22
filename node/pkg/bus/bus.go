package bus

// message bus will be passed as parameter to modules that need to communicate with each other

type Message struct {
	From    string
	To      string
	Content MessageContent
}

type MessageContent struct {
	Command string
	Args    map[string]interface{}
}

type MessageBus struct {
	channels map[string]chan Message
}

func NewMessageBus() *MessageBus {
	return &MessageBus{
		channels: make(map[string]chan Message),
	}
}

func (mb *MessageBus) Subscribe(id string, buffer int) <-chan Message {
	ch := make(chan Message, buffer)
	mb.channels[id] = ch
	return mb.channels[id]
}

func (mb *MessageBus) Publish(msg Message) {
	if ch, ok := mb.channels[msg.To]; ok {
		ch <- msg
	}
}
