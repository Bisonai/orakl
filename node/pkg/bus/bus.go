package bus

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

func (mb *MessageBus) Publish(msg Message) {
	if ch, ok := mb.channels[msg.To]; ok {
		ch <- msg
	}
}
