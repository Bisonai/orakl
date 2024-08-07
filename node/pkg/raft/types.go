package raft

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type MessageType string
type RoleType string

const (
	Heartbeat          MessageType = "heartbeat"
	RequestVote        MessageType = "requestVote"
	ReplyVote          MessageType = "replyVote"
	AppendEntries      MessageType = "appendEntries"
	ReplyAppendEntries MessageType = "replyAppendEntries"

	Leader    RoleType = "leader"
	Candidate RoleType = "candidate"
	Follower  RoleType = "follower"

	MessageCleanupInterval = 10 * time.Second
	MessageTTL             = 30 * time.Second
)

type Message struct {
	Type     MessageType     `json:"type"`
	SentFrom string          `json:"sentFrom"`
	Data     json.RawMessage `json:"data"`
}

func (m *Message) Hash() (*string, error) {
	hash := sha256.New()
	if _, err := hash.Write([]byte(m.Type)); err != nil {
		return nil, err
	}
	if _, err := hash.Write([]byte(m.SentFrom)); err != nil {
		return nil, err
	}
	if _, err := hash.Write(m.Data); err != nil {
		return nil, err
	}

	hashString := hex.EncodeToString(hash.Sum(nil))
	return &hashString, nil
}

type RequestVoteMessage struct {
	Term int `json:"term"`
}

type HeartbeatMessage struct {
	LeaderID string `json:"leaderID"`
	Term     int    `json:"term"`
}

type ReplyRequestVoteMessage struct {
	VoteGranted bool   `json:"voteGranted"`
	LeaderID    string `json:"leaderID"`
}

type Raft struct {
	Host  host.Host
	Ps    *pubsub.PubSub
	Topic *pubsub.Topic

	Role          RoleType
	VotedFor      string
	LeaderID      string
	VotesReceived int
	Term          int
	Mutex         sync.Mutex

	HeartbeatTicker  *time.Ticker
	ElectionTimer    *time.Timer
	Resign           chan interface{}
	MessageBuffer    chan Message
	HeartbeatTimeout time.Duration

	LeaderJobTimeout    time.Duration
	LeaderJobTicker     *time.Ticker
	HandleCustomMessage func(context.Context, Message) error
	LeaderJob           func(context.Context) error
}
