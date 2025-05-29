package bus

import (
	"strconv"
	"sync"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"github.com/rs/zerolog/log"
)

// message bus will be passed as parameter to modules that need to communicate with each other

type Message struct {
	From     string
	To       string
	Content  MessageContent
	Response chan MessageResponse
}

type MessageContent struct {
	Command string
	Args    map[string]any
}

type MessageResponse struct {
	Success bool
	Args    map[string]any
}

type MessageBus struct {
	channels  map[string]chan Message
	msgBuffer int
	sync.RWMutex
}

func New(bufferSize int) *MessageBus {
	return &MessageBus{
		channels:  make(map[string]chan Message),
		msgBuffer: bufferSize,
	}
}

func (mb *MessageBus) Subscribe(id string) <-chan Message {
	mb.Lock()
	defer mb.Unlock()
	ch := make(chan Message, mb.msgBuffer)
	mb.channels[id] = ch
	return ch
}

func (mb *MessageBus) Publish(msg Message) error {
	mb.RLock()
	defer mb.RUnlock()
	ch, ok := mb.channels[msg.To]
	if !ok {
		return errorSentinel.ErrBusChannelNotFound
	}
	select {
	case ch <- msg:
		return nil
	default:
		return errorSentinel.ErrBusMsgPublishFail
	}
}

func ParseInt64MsgParam(msg Message, param string) (int64, error) {
	rawId, ok := msg.Content.Args[param]
	if !ok {
		return 0, errorSentinel.ErrBusParamNotFound
	}

	idPayload, ok := rawId.(string)
	if !ok {
		return 0, errorSentinel.ErrBusConvertParamFail
	}

	id, err := strconv.ParseInt(idPayload, 10, 64)
	if err != nil {
		return 0, errorSentinel.ErrBusParseParamFail
	}

	return id, nil
}

func ParseInt32MsgParam(msg Message, param string) (int32, error) {
	rawId, ok := msg.Content.Args[param]
	if !ok {
		return 0, errorSentinel.ErrBusParamNotFound
	}

	idPayload, ok := rawId.(string)
	if !ok {
		return 0, errorSentinel.ErrBusConvertParamFail
	}

	id, err := strconv.ParseInt(idPayload, 10, 32)
	if err != nil {
		return 0, errorSentinel.ErrBusParseParamFail
	}

	return int32(id), nil
}

func ParseStringMsgParam(msg Message, param string) (string, error) {
	raw, ok := msg.Content.Args[param]
	if !ok {
		return "", errorSentinel.ErrBusParamNotFound
	}

	payload, ok := raw.(string)
	if !ok {
		return "", errorSentinel.ErrBusConvertParamFail
	}

	return payload, nil
}

func HandleMessageError(err error, msg Message, logMessage string) {
	log.Error().Err(err).Msg(logMessage)
	msg.Response <- MessageResponse{Success: false, Args: map[string]any{"error": err.Error()}}
}
