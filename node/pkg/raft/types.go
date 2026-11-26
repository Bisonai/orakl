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

	// above the longest leader silence seen on mainnet (<7.5 s over 24 h,
	// 99.99% <2.5 s), so a stall doesn't count a live peer out
	DefaultPeerSilenceTimeout = 10 * time.Second
	DefaultLeaderJobWatchdog  = 10 * time.Second
	// an unhealthy leader (CanLead) resigns a feed at most once per this, so
	// a cluster whose nodes are all unhealthy settles instead of handing feeds
	// around
	DefaultHealthResignBackoff = 5 * time.Minute
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

	// set through setRole, which keeps IsLeader in sync (under Mutex)
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

	// a subscribed peer silent this long isn't counted as a voter, so a peer
	// that hangs while connected can't block every election
	PeerSilenceTimeout time.Duration
	// a leader whose LeaderJob hasn't succeeded this long resigns
	LeaderJobWatchdog time.Duration
	// a follower whose leader left the topic campaigns this long after, if it
	// is still gone (peer_leave.go); 0 is the minimum election timeout plus
	// a heartbeat interval, negative never
	PeerLeaveGrace time.Duration

	// no campaign before this, after resigning a stuck term; under Mutex
	campaignAfter time.Time

	// CanLead, if set before Run, gates leadership on this node's health (the
	// in-node reporter, which submits only the feeds this node leads): while
	// it returns false, a leader resigns to a live peer and a follower holds
	// off campaigning (health.go). nil: always healthy.
	CanLead func() bool
	// an unhealthy leader resigns at most once per this; 0 is
	// DefaultHealthResignBackoff
	HealthResignBackoff time.Duration
	// last resign for health, ns since leaderClockEpoch; 0 never
	healthResignedAt atomic.Int64
	// no live-peer check for a health resign before this, same unit
	healthPeerCheckAt atomic.Int64
	// an unhealthy follower campaigns only from this on; zero until it first
	// wants to after following a leader; under Mutex
	unhealthyCampaignAfter time.Time

	// message types only a leader sends, besides heartbeats; one from the
	// current leader holds off votes (hearsLeaderOtherThan) and, like a
	// heartbeat, the election timer (heardLeaderSinceTimerArmed). Set before Run.
	LeaderMessageTypes map[MessageType]bool

	// a leader skips heartbeats that a recent leader message makes redundant
	// (sparse_heartbeat.go); NewRaftNode sets it from P2P_SPARSE_HEARTBEAT.
	// Set before Run.
	SparseHeartbeat bool
	// last heartbeat or LeaderMessageTypes message this node published, ns
	// since leaderClockEpoch; only with SparseHeartbeat
	leaderSentAt atomic.Int64
	// last event that wants a heartbeat every tick (stampWantHeartbeats), same
	// unit; only with SparseHeartbeat
	wantHeartbeatsAt atomic.Int64

	// peer.ID -> *atomic.Int64, ns since leaderClockEpoch of its last message
	peerSeen sync.Map
	// term a deferred win recount is pending for (handleReplyVote); under Mutex
	recountTerm int
	// when this node last granted its vote to another peer; under Mutex
	votedAt time.Time
	// when the election timer was last armed, ns since leaderClockEpoch; under Mutex
	timerArmedAt int64

	// leader loops and the leader jobs they started, for Run to wait on
	leaderLoops goroutineCount

	// atomic mirror of LeaderID for the message path, which must not take Mutex
	// for metrics; nil when this node is the leader or there is none
	leaderClock atomic.Pointer[leaderClock]

	// atomic mirror of Role == Leader, for IsLeader (setRole)
	leading atomic.Bool

	// closed when this node starts following another leader (followLeader),
	// for AwaitLeader; nil while nobody waits; under Mutex
	leaderFollowed chan struct{}
}

// leaderClock tracks when a leader's last message arrived. It lives as long as
// that peer is the leader, so a gap is never measured across a leader change.
type leaderClock struct {
	id         string
	term       int          // term this peer last became the leader at, under Raft.Mutex
	followedAt int64        // when it became the leader, ns since leaderClockEpoch; fixed
	lastSeen   atomic.Int64 // ns since leaderClockEpoch
	// last heartbeat or LeaderMessageTypes message, same unit: leader
	// stickiness (hearsLeaderOtherThan)
	activeAt atomic.Int64
	// when its latest event on the topic, a leave, came (watchPeerLeaves),
	// same unit; 0 while it's on the topic
	leftAt atomic.Int64
}
