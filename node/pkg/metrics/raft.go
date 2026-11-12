package metrics

// Raft failover metrics, aggregated over all feeds: each feed is its own Raft
// group, and a per-feed label would be too many series on this path.

import (
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Reasons an election starts.
const (
	// the election timer ran out: the leader missed heartbeats
	RaftElectionReasonTimeout = "timeout"
	// the leader's connection dropped (it left the feed's topic, not connected),
	// including a node shutdown, which closes its host at once
	RaftElectionReasonDisconnect = "disconnect"
	// the leader left the feed's topic while still connected: its aggregator
	// stopped, or a shutdown's unsubscribe arrived before the disconnect
	RaftElectionReasonShutdown = "shutdown"
)

// raftSilenceBuckets are fine around the election threshold (~1.5-3 s at the 100 ms
// heartbeat), with a tail for gaps that span an election: a peer winning again only
// shows in the leader silence histogram.
var raftSilenceBuckets = []float64{0.05, 0.1, 0.15, 0.2, 0.3, 0.5, 0.75, 1, 1.5, 2, 2.5, 3, 4, 5, 7.5, 10, 15, 30, 60}

var (
	raftLeaderSilence = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    metricnames.RaftLeaderSilence,
		Help:    "Time since the previous message from the current leader of a feed, observed on each message from that leader",
		Buckets: raftSilenceBuckets,
	})
	raftLeaderChangeSilence = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    metricnames.RaftLeaderChangeSilence,
		Help:    "Time since the last message from a feed's leader, observed when another peer or this node replaces it",
		Buckets: raftSilenceBuckets,
	})
	raftElectionsStarted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.RaftElectionsStarted,
		Help: "Raft elections started by this node, by reason",
	}, []string{metricnames.ReasonLabel})
	raftHeartbeatsSkipped = promauto.NewCounter(prometheus.CounterOpts{
		Name: metricnames.RaftHeartbeatsSkipped,
		Help: "Heartbeats this node skipped as leader because a recent leader message made them redundant (P2P_SPARSE_HEARTBEAT)",
	})
)

func init() {
	// exported at 0, so the first election shows up in rate() and increase()
	for _, reason := range []string{RaftElectionReasonTimeout, RaftElectionReasonDisconnect, RaftElectionReasonShutdown} {
		raftElectionsStarted.WithLabelValues(reason)
	}
}

// RecordRaftLeaderSilence observes one gap between messages from a feed's leader.
func RecordRaftLeaderSilence(gap time.Duration) {
	raftLeaderSilence.Observe(gap.Seconds())
}

// RecordRaftLeaderChangeSilence observes the silence of a feed's leader at the
// moment it was replaced.
func RecordRaftLeaderChangeSilence(gap time.Duration) {
	raftLeaderChangeSilence.Observe(gap.Seconds())
}

// RecordRaftElectionStarted counts one election started for reason.
func RecordRaftElectionStarted(reason string) {
	raftElectionsStarted.WithLabelValues(reason).Inc()
}

// RecordRaftHeartbeatSkipped counts one heartbeat a leader skipped.
func RecordRaftHeartbeatSkipped() {
	raftHeartbeatsSkipped.Inc()
}
