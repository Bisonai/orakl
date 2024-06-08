package raft

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/utils/set"
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
	ReplyHeartbeat     MessageType = "replyHeartbeat"

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

type ReplyHeartbeatMessage struct {
	Term int `json:"term"`
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
	LeaderJob           func() error

	PrevPeers set.Set[string]
	Peers     set.Set[string]
}
