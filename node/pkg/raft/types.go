package raft

import (
	"encoding/json"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type MessageType string
type RoleType string

const (
	Heartbeat MessageType = "heartbeat"
	// ReplyHeartbeat     MessageType = "replyHeartbeat"
	RequestVote        MessageType = "requestVote"
	ReplyVote          MessageType = "replyVote"
	AppendEntries      MessageType = "appendEntries"
	ReplyAppendEntries MessageType = "replyAppendEntries"

	Leader    RoleType = "leader"
	Candidate RoleType = "candidate"
	Follower  RoleType = "follower"
)

type Message struct {
	Type     MessageType     `json:"type"`
	SentFrom string          `json:"sentFrom"`
	Data     json.RawMessage `json:"data"`
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
}

type Node interface {
	GetHost() host.Host
	GetPubSub() *pubsub.PubSub
	GetTopic() *pubsub.Topic
	GetSub() *pubsub.Subscription

	HandleCustomMessage(Message) error

	// define regular job run by leader
	GetLeaderJobTimeout() *time.Duration
	GetLeaderJobTicker() *time.Ticker
	SetLeaderJobTicker(*time.Duration) error
	LeaderJob() error

	// define regular job run by every node
	GetJobTimeout() *time.Duration
	GetJobTicker() *time.Ticker
	SetJobTicker(*time.Duration) error
	Job() error
}
