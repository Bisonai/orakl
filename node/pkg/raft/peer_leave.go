package raft

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"bisonai.com/orakl/node/pkg/metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

// Electing when the leader leaves the topic.
//
// libp2p tells a feed's Raft that a peer is gone through the topic's own peer
// events: pubsub drops a peer from every topic, and emits PeerLeave, when its
// connection closes (it notices on the first failed stream read, as soon as the
// connection does; a node shutdown closes its host for this) or when the peer
// unsubscribes (an aggregator stop). The same step removes it from ListPeers, so it stops counting as a
// voter at once (voterCounts) and a lone survivor wins on its own vote. Each
// Raft watches its own topic, so the libp2p layer needs no list of Rafts and no
// package imports another.
//
// A peer that is back on the topic by the end of PeerLeaveGrace (plus up to
// half a minimum election timeout of jitter) is ignored: a
// connection the reconnect worker restores, or that pubsub reopens a stream on,
// causes no election. One that comes back later finds a successor at a higher
// term and follows it; the cooldown keeps flaps from electing again.

// watchPeerLeaves runs until ctx is done or the topic has no event handler.
func (r *Raft) watchPeerLeaves(ctx context.Context, events *pubsub.TopicEventHandler) {
	defer events.Cancel()
	grace := r.peerLeaveGrace()
	// per peer, the number of events seen: a leave acts only if it is still
	// the peer's latest, so a flap's earlier leave can't fire during a later one
	seen := map[peer.ID]*atomic.Uint64{}
	for {
		event, err := events.NextPeerEvent(ctx)
		if err != nil {
			return
		}
		if event.Peer == r.Host.ID() {
			continue
		}
		count, ok := seen[event.Peer]
		if !ok {
			count = &atomic.Uint64{}
			seen[event.Peer] = count
		}
		seq := count.Add(1)
		r.markDeparted(event.Peer, event.Type == pubsub.PeerLeave)
		if event.Type == pubsub.PeerJoin {
			r.wantHeartbeatsFromJoin()
		}
		if event.Type != pubsub.PeerLeave {
			continue
		}
		left, leftAt := event.Peer, time.Now()
		// at the leave, not after the grace: a reconnect in between would
		// make a disconnect look like an unsubscribe
		reason := metrics.RaftElectionReasonDisconnect
		if r.Host.Network().Connectedness(left) == network.Connected {
			reason = metrics.RaftElectionReasonShutdown // it unsubscribed
		}
		// jittered, so with 3+ nodes the followers of a leader that left
		// don't all campaign at once and split the vote
		wait := grace + time.Duration(rand.Int63n(int64(r.minElectionTimeout()/2)+1))
		time.AfterFunc(wait, func() {
			if count.Load() == seq {
				r.handlePeerLeft(ctx, left, leftAt, reason)
			}
		})
	}
}

// A leader that left the topic is gone (isGone) from its leave on, until it
// rejoins (PeerJoin) or is heard again past any backlog (reviveIfHeard).
// Whatever of it still arrives in between was sent before it left: a backlog
// delivering that late must not hold off the vote for its successor
// (hearsLeaderOtherThan), end the successor's candidacy (handleHeartbeat) or
// hold off the election timer, also on a retry after a split vote (#3012).
// Kept per peer, not on the leader clock, so a leave counts even when this
// node only follows that leader afterwards, from a late heartbeat. A flap
// stays harmless: no leave causes an election before the grace, and a rejoin
// within it clears the leave.

// departedRetention is how long a departed peer that isn't the leader this
// node follows is remembered, so peers that never return (a restart takes a
// new peer ID) don't pile up.
const departedRetention = 10 * time.Minute

// markDeparted records a peer's leave of the topic, or forgets it on a join.
func (r *Raft) markDeparted(id peer.ID, left bool) {
	key := id.String()
	if !left {
		r.departed.Delete(key)
		return
	}
	now := int64(time.Since(leaderClockEpoch))
	r.departed.Store(key, now)
	clock := r.leaderClock.Load()
	r.departed.Range(func(k, v any) bool {
		if (clock == nil || k.(string) != clock.id) && time.Duration(now-v.(int64)) > departedRetention {
			r.departed.CompareAndDelete(k, v)
		}
		return true
	})
}

// isGone reports whether the peer left the topic and isn't back (markDeparted,
// reviveIfHeard).
func (r *Raft) isGone(id string) bool {
	_, gone := r.departed.Load(id)
	return gone
}

// reviveIfHeard forgets a departed peer's leave on a message it signed that
// arrives past departedBacklog: no backlog lasts that long, so a peer still
// reaches it and relays it (only the direct link dropped), and it is heard
// again: a live leader isn't ignored for as long as the link is down.
func (r *Raft) reviveIfHeard(id string) {
	leftAt, gone := r.departed.Load(id)
	if !gone {
		return
	}
	if time.Duration(int64(time.Since(leaderClockEpoch))-leftAt.(int64)) > r.departedBacklog() {
		r.departed.CompareAndDelete(id, leftAt)
	}
}

// departedBacklog bounds how late a departed peer's messages are still taken
// for a backlog: past the election cooldown and a maximum election timeout,
// the split-vote retry after its leave (#3043) has had its time.
func (r *Raft) departedBacklog() time.Duration {
	return r.CooldownPeriod + 2*r.minElectionTimeout()
}

// peerLeaveGrace is PeerLeaveGrace, or when unset the minimum election timeout
// plus a heartbeat interval: by then the leader's last message is past the
// stickiness window (hearsLeaderOtherThan) on every peer, so none refuses the
// first candidate. Negative disables electing on a leave.
func (r *Raft) peerLeaveGrace() time.Duration {
	if r.PeerLeaveGrace == 0 {
		return r.minElectionTimeout() + r.HeartbeatTimeout
	}
	return r.PeerLeaveGrace
}

// handlePeerLeft acts on a peer that left the topic a grace ago and is still
// gone. A follower of it campaigns at once, skipping the missed heartbeats; a
// candidate recounts, as the vote it waited for won't come.
func (r *Raft) handlePeerLeft(ctx context.Context, left peer.ID, leftAt time.Time, reason string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if ctx.Err() != nil || r.onTopic(left) {
		return
	}
	// dropped and already reconnected: pubsub is still re-exchanging
	// subscriptions, so this is a flap, not a lost leader
	if reason == metrics.RaftElectionReasonDisconnect && r.Host.Network().Connectedness(left) == network.Connected {
		return
	}
	switch r.Role {
	case Follower:
		if r.LeaderID != left.String() {
			return
		}
		// another follower got there first and has this node's vote, granted
		// around the leave (its RequestVote can be handled before this node
		// reads its own leave event): a campaign of its own would only depose
		// it (3+ nodes). A vote from an earlier election doesn't count. The
		// timer still runs if that candidate fails.
		if r.VotedFor != "" && r.VotedFor != r.GetHostId() && r.votedAt.After(leftAt.Add(-r.minElectionTimeout())) {
			return
		}
		log.Info().Str("Player", "Raft").Str("topic", r.Topic.String()).Str("leader", left.String()).Str("reason", reason).Msg("leader left the topic, starting election")
		r.campaign(ctx, reason)
	case Candidate:
		if subscribed, _ := r.voterCounts(); r.VotesReceived >= votesNeeded(subscribed) {
			log.Info().Str("Player", "Raft").Str("topic", r.Topic.String()).Str("peer", left.String()).Msg("peer left the topic, enough votes without it")
			r.becomeLeader(ctx)
		}
	}
}

// onTopic reports whether pubsub lists the peer on this node's topic.
func (r *Raft) onTopic(id peer.ID) bool {
	for _, listed := range r.Ps.ListPeers(r.Topic.String()) {
		if listed == id {
			return true
		}
	}
	return false
}
