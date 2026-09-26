package raft

import (
	"context"
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
	BatchHeartbeat     MessageType = "batchHeartbeat"
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

// BatchHeartbeatMessage is one combined heartbeat covering every feed this node
// currently leads: Terms maps a feed key to that feed's Raft term. The leader id
// is carried once by the enclosing Message.SentFrom, so it is not repeated per
// feed. It rides the shared control topic instead of the ~150 per-feed topics,
// collapsing ~1,500 heartbeat msgs/s into ~10-20/s (issue #2558).
//
// The key is the feed's stable name (config.Name), NOT the local config.id: ids
// are node-local serial primary keys assigned by each node's own insert history
// (configs are upserted on conflict(name) and never carry a shared id), so an id
// means different feeds on different nodes. Per-feed data topics key on the name
// for the same reason; the batch must too or a follower would apply a leader's
// heartbeat to the wrong feed.
type BatchHeartbeatMessage struct {
	Terms map[string]int `json:"terms"`
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
	MessageBuffer    chan *pubsub.Message
	HeartbeatTimeout time.Duration

	LeaderJobTimeout    time.Duration
	LeaderJobTicker     *time.Ticker
	HandleCustomMessage func(context.Context, Message) error
	LeaderJob           func(context.Context) error

	MissedHeartbeats int

	CooldownPeriod   time.Duration
	LastElectionTime time.Time
}
