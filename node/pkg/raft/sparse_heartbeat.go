package raft

import (
	"os"
	"strconv"
	"strings"
	"time"
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
// Every peer must count leader messages as liveness (#2813, v0.0.32+): an
// older follower times out a leader whose heartbeats are 1 s apart.

// SparseHeartbeatEnvVar enables sparse heartbeats when set to a true value
// (strconv.ParseBool). Unset or invalid is off: every tick sends a heartbeat.
const SparseHeartbeatEnvVar = "P2P_SPARSE_HEARTBEAT"

// SparseHeartbeatEnabled reports whether SparseHeartbeatEnvVar is set to true.
func SparseHeartbeatEnabled() bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(SparseHeartbeatEnvVar)))
	return err == nil && enabled
}

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

// skipHeartbeat reports whether the leader loop can skip this heartbeat tick.
// Lock-free: it reads the stamps PublishMessage leaves.
func (r *Raft) skipHeartbeat() bool {
	if !r.SparseHeartbeat {
		return false
	}
	// half a tick of slack: heartbeats sit on ticks, so the forced one goes
	// out on the tick a full maxHeartbeatGap after the last
	return since(&r.leaderSentAt) < r.sparseHeartbeatWindow() &&
		since(&r.heartbeatSentAt) < r.maxHeartbeatGap()-r.HeartbeatTimeout/2
}

// stampLeaderSent records a published heartbeat or leader message, at sentAt
// (ns since leaderClockEpoch, taken before publishing, so a slow publish only
// makes the next heartbeat earlier). Only with SparseHeartbeat on.
func (r *Raft) stampLeaderSent(msgType MessageType, sentAt int64) {
	if !r.SparseHeartbeat {
		return
	}
	if msgType == Heartbeat {
		storeLater(&r.heartbeatSentAt, sentAt)
	} else if !r.LeaderMessageTypes[msgType] {
		return
	}
	storeLater(&r.leaderSentAt, sentAt)
}
