package raft

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Sparse heartbeats (#2818, phase 2 of #2747), behind P2P_SPARSE_HEARTBEAT.
//
// A leader skips a heartbeat tick when it published a heartbeat or a
// LeaderMessageTypes message (trigger, price fix) less than
// sparseHeartbeatWindow ago, unless its last heartbeat is about
// maxHeartbeatGap old. Followers count those leader messages as liveness for
// the election timer and for stickiness, so they carry no less than before.
//
// Stickiness rule: hearsLeaderOtherThan holds for minElectionTimeout (500 ms)
// after a leader message. With skipping, the longest gap between two leader
// messages the leader publishes is below sparseHeartbeatWindow plus one tick:
// 250 + 100 = 350 ms, for any AggregateInterval, because heartbeats fill any
// gap rounds leave. That leaves 150 ms for delivery jitter, where today's
// gap is one tick (100 ms). The election timer (three expiries of 500-1000 ms),
// PeerLeaveGrace and shutdown paths don't depend on heartbeat spacing, so
// failover is as fast as before.
//
// Heartbeats are the only leader message with a term, and the one a follower
// without a leader (just joined, or a stalled candidate at a higher term)
// follows. maxHeartbeatGap (1 s) is below the earliest such node campaigns
// against the leader: MaxMissedHeartbeats+1 expiries of at least 500 ms.
//
// A leader sends a heartbeat every tick for maxHeartbeatGap after anything
// that shows a peer needs one soon (stampWantHeartbeats): another leader's
// message (two leaders resolve only through a heartbeat, which carries the
// term), a candidate's RequestVote, or a peer joining the topic. A dual
// leadership, a stalled candidate or a joiner then gets the term as fast as
// with the flag off, and a lost forced heartbeat costs at most one election.
//
// Every peer must count leader messages as liveness (#2813, v0.0.32+): an
// older follower times out a leader whose heartbeats are 1 s apart.

// SparseHeartbeatEnvVar enables sparse heartbeats when set to a true value
// (strconv.ParseBool). Unset or invalid is off: every tick sends a heartbeat.
const SparseHeartbeatEnvVar = "P2P_SPARSE_HEARTBEAT"

// neverSent is a stamp long enough ago that nothing is skipped on it, for a
// term that has published no heartbeat yet: 0 is recent in a process's first
// second.
const neverSent = -1 << 62

// SparseHeartbeatEnabled reports whether SparseHeartbeatEnvVar is set to true.
func SparseHeartbeatEnabled() bool {
	value := strings.TrimSpace(os.Getenv(SparseHeartbeatEnvVar))
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		// once: every feed's Raft reads it
		warnInvalidSparseHeartbeat.Do(func() {
			log.Warn().Str("Player", "Raft").Str(SparseHeartbeatEnvVar, value).Msg("invalid value, sparse heartbeats off")
		})
		return false
	}
	return enabled
}

var warnInvalidSparseHeartbeat sync.Once

// sparseHeartbeatWindow is how recent a leader message must be for a heartbeat
// tick to be skipped: half the stickiness window, 250 ms at the default 100 ms
// heartbeat, so a feed with no rounds sends a heartbeat every third tick.
func (r *Raft) sparseHeartbeatWindow() time.Duration {
	return r.minElectionTimeout() / 2
}

// maxHeartbeatGap is the longest a leader goes without a heartbeat while
// skipping: the maximum election timeout, 1 s at the default heartbeat.
func (r *Raft) maxHeartbeatGap() time.Duration {
	return 2 * r.minElectionTimeout()
}

// skipHeartbeat reports whether the leader loop can skip this heartbeat tick,
// given when its term last published one (heartbeatAt). Lock-free.
func (r *Raft) skipHeartbeat(heartbeatAt int64) bool {
	if !r.SparseHeartbeat {
		return false
	}
	// half a tick of slack: heartbeats sit on ticks, so the forced one goes
	// out on the tick a full maxHeartbeatGap after the last
	return since(&r.leaderSentAt) < r.sparseHeartbeatWindow() &&
		sinceStamp(heartbeatAt) < r.maxHeartbeatGap()-r.HeartbeatTimeout/2 &&
		since(&r.wantHeartbeatsAt) >= r.maxHeartbeatGap()
}

// sinceStamp is since for a plain stamp.
func sinceStamp(at int64) time.Duration {
	return time.Since(leaderClockEpoch) - time.Duration(at)
}

// stampLeaderSent records a heartbeat or LeaderMessageTypes message published
// at (ns since leaderClockEpoch, taken before publishing, so a slow publish
// only makes the next heartbeat earlier). Only with SparseHeartbeat on.
func (r *Raft) stampLeaderSent(at int64) {
	if r.SparseHeartbeat {
		storeLater(&r.leaderSentAt, at)
	}
}

// stampLeaderMessageSent is stampLeaderSent for a message PublishMessage
// published at sentAt, if it is of a LeaderMessageTypes type.
func (r *Raft) stampLeaderMessageSent(msgType MessageType, sentAt time.Time) {
	if r.SparseHeartbeat && r.LeaderMessageTypes[msgType] {
		r.stampLeaderSent(int64(sentAt.Sub(leaderClockEpoch)))
	}
}

// stampWantHeartbeats makes a leader send a heartbeat every tick for
// maxHeartbeatGap, on a message signed by author that shows a peer needs the
// term or the leader soon:
//   - a valid heartbeat, trigger or price fix from another peer: while this
//     node leads, another leader, which only a heartbeat makes resign;
//   - a RequestVote from another peer: a candidate, which a heartbeat ends.
//
// wantHeartbeatsFromJoin covers a peer joining the topic. A deposed leader's
// late round counts too, which costs at most maxHeartbeatGap of heartbeats on
// every tick after a failover. Only with SparseHeartbeat on; a follower's
// stamp is unused.
func (r *Raft) stampWantHeartbeats(msgType MessageType, author string) {
	if !r.SparseHeartbeat || author == r.GetHostId() {
		return
	}
	if msgType == Heartbeat || msgType == RequestVote || r.LeaderMessageTypes[msgType] {
		r.wantHeartbeatsNow()
	}
}

// wantHeartbeatsFromJoin is stampWantHeartbeats for a peer that joined the
// topic: it has no leader until a heartbeat, and campaigns if it misses the
// forced one.
func (r *Raft) wantHeartbeatsFromJoin() {
	if r.SparseHeartbeat {
		r.wantHeartbeatsNow()
	}
}

func (r *Raft) wantHeartbeatsNow() {
	storeLater(&r.wantHeartbeatsAt, int64(time.Since(leaderClockEpoch)))
}
