package raft

import (
	"time"

	"github.com/rs/zerolog/log"
)

// Leadership gated on node health (CanLead, #2914). With the in-node reporter
// only a feed's leader submits it, so a leader that can't submit stops the
// feed on-chain while its rounds and heartbeats keep peers following it.
//
// An unhealthy leader resigns to a live peer like a stuck one
// (resignStuckLeader), and an unhealthy follower leaves an election to its
// peers for as long as a healthy one takes to win it. Neither gives the feed
// up when nothing could take over: with no live peer it keeps leading, and
// with only unhealthy peers each resigns a feed at most once per
// HealthResignBackoff and campaigns after the hold-off, so the feed keeps a
// leader (and its rounds) instead of moving around.

// unhealthy reports whether CanLead is set and returns false.
func (r *Raft) unhealthy() bool {
	return r.CanLead != nil && !r.CanLead()
}

func (r *Raft) healthResignBackoff() time.Duration {
	if r.HealthResignBackoff > 0 {
		return r.HealthResignBackoff
	}
	return DefaultHealthResignBackoff
}

// resignUnhealthyLeader resigns the term resign belongs to if this node is
// unhealthy, a live peer can take over, and it hasn't resigned for health
// within HealthResignBackoff. Like resignStuckLeader it holds off campaigning,
// so a peer takes the term rather than this node winning it back. Reports
// whether the loop must stop. Runs on every heartbeat tick: lock-free unless
// it resigns or checks for a live peer, which it does at most once per
// minimum election timeout.
func (r *Raft) resignUnhealthyLeader(resign <-chan interface{}) bool {
	if r.CanLead == nil {
		return false
	}
	if r.CanLead() {
		return false
	}
	now := int64(time.Since(leaderClockEpoch))
	if at := r.healthResignedAt.Load(); at != 0 && time.Duration(now-at) < r.healthResignBackoff() {
		return false
	}
	// a lone node would otherwise take Mutex every tick (#2749)
	if now < r.healthPeerCheckAt.Load() {
		return false
	}

	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if resigned(resign) {
		return true
	}
	if _, live := r.voterCounts(); live == 0 {
		r.healthPeerCheckAt.Store(now + int64(r.minElectionTimeout()))
		return false
	}
	log.Warn().Str("Player", "Raft").Msg("node unhealthy, resigning")
	r.healthResignedAt.Store(int64(time.Since(leaderClockEpoch)))
	r.ResignLeader()
	r.MissedHeartbeats = 0
	r.campaignAfter = time.Now().Add(2 * r.maxElectionDetection())
	// one hold-off for this leaderless spell, not campaignAfter's plus another
	r.unhealthyCampaignAfter = r.campaignAfter
	return true
}

// holdOffUnhealthyCampaign reports whether an unhealthy node must not campaign
// yet: for twice the longest a peer takes to notice a missing leader and
// campaign, from its first try after following a leader, so a healthy peer
// wins. With no live peer it campaigns at once. The caller holds Mutex.
func (r *Raft) holdOffUnhealthyCampaign() bool {
	if !r.unhealthy() {
		r.unhealthyCampaignAfter = time.Time{}
		return false
	}
	if _, live := r.voterCounts(); live == 0 {
		return false
	}
	now := time.Now()
	if r.unhealthyCampaignAfter.IsZero() {
		r.unhealthyCampaignAfter = now.Add(2 * r.maxElectionDetection())
	}
	return now.Before(r.unhealthyCampaignAfter)
}
