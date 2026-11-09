package raft

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
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

	MaxMissedHeartbeats   = 2
	DefaultCooldownPeriod = 3 * time.Second
)

type Message struct {
	Type      MessageType     `json:"type"`
	SentFrom  string          `json:"sentFrom"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
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

	ElectionTimer    *time.Timer
	Resign           chan interface{}
	MessageBuffer    chan *pubsub.Message
	HeartbeatTimeout time.Duration

	LeaderJobTimeout    time.Duration
	HandleCustomMessage func(context.Context, Message) error
	LeaderJob           func(context.Context) error

	MissedHeartbeats int

	CooldownPeriod   time.Duration
	LastElectionTime time.Time

	// atomic mirror of LeaderID for the message path, which must not take Mutex
	// for metrics; nil when this node is the leader or there is none
	leaderClock atomic.Pointer[leaderClock]
}

// leaderClock tracks when a leader's last message arrived. It lives as long as
// that peer is the leader, so a gap is never measured across a leader change.
type leaderClock struct {
	id       string
	lastSeen atomic.Int64 // ns since leaderClockEpoch
}
