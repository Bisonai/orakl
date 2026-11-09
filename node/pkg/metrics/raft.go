package metrics

// Raft failover metrics, aggregated over all feeds: each feed is its own Raft
// group, and a per-feed label would be too many series on this path.

import (
	"time"

	metricnames "bisonai.com/miko/node/pkg/metrics/names"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RaftElectionReasonTimeout is an election started by the election timer.
const RaftElectionReasonTimeout = "timeout"

var (
	raftLeaderSilence = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: metricnames.RaftLeaderSilence,
		Help: "Time since the previous message from the current leader of a feed, observed on each message from that leader",
		// fine around the election threshold (~1.5-3 s at the 100 ms heartbeat)
		Buckets: []float64{0.05, 0.1, 0.15, 0.2, 0.3, 0.5, 0.75, 1, 1.5, 2, 2.5, 3, 4, 5, 7.5, 10},
	})
	raftLeaderChangeSilence = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: metricnames.RaftLeaderChangeSilence,
		Help: "Time since the last message from a feed's leader, observed when another peer or this node replaces it",
		// wider than the leader silence buckets: this tail includes the election
		Buckets: []float64{0.05, 0.1, 0.15, 0.2, 0.3, 0.5, 0.75, 1, 1.5, 2, 2.5, 3, 4, 5, 7.5, 10, 15, 30, 60},
	})
	raftElectionsStarted = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: metricnames.RaftElectionsStarted,
		Help: "Raft elections started by this node, by reason",
	}, []string{metricnames.ReasonLabel})
)

func init() {
	// exported at 0, so the first election shows up in rate() and increase()
	raftElectionsStarted.WithLabelValues(RaftElectionReasonTimeout)
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
