package raft

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testLeaderInterval = 20 * time.Millisecond
	testLoopSettle     = 5 * testLeaderInterval
)

// newLeaderLoopNode returns a node that counts leader job runs and has the short
// tickers these tests wait on.
func newLeaderLoopNode(ctx context.Context, t *testing.T) (*Raft, *atomic.Int64) {
	t.Helper()
	node := newPublishTestNode(ctx, t)
	node.HeartbeatTimeout = testLeaderInterval
	node.LeaderJobTimeout = testLeaderInterval
	node.startElectionTimer()

	jobs := &atomic.Int64{}
	node.LeaderJob = func(context.Context) error {
		jobs.Add(1)
		return nil
	}
	return node, jobs
}

// assertLoopStopped checks that no leader job starts anymore. Only a running
// loop starts them, so its ticker must be stopped.
func assertLoopStopped(t *testing.T, jobs *atomic.Int64) {
	t.Helper()
	time.Sleep(testLoopSettle)
	before := jobs.Load()
	time.Sleep(testLoopSettle)
	assert.Equal(t, before, jobs.Load(), "leader loop still running")
}

func TestLeaderLoopStopsOnResignDuringFirstJob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)
	started := make(chan struct{})
	release := make(chan struct{})
	countJob := node.LeaderJob
	var first atomic.Bool
	node.LeaderJob = func(ctx context.Context) error {
		if first.CompareAndSwap(false, true) {
			close(started)
			<-release
		}
		return countJob(ctx)
	}

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()

	<-started
	node.Mutex.Lock()
	node.ResignLeader()
	node.Mutex.Unlock()
	close(release)

	assertLoopStopped(t, jobs)
}

func TestLeaderLoopStopsOnResignMidIteration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()

	require.Eventually(t, func() bool { return jobs.Load() >= 2 }, 5*time.Second, testLeaderInterval, "loop not running")

	// sendHeartbeat takes the mutex, so holding it past a tick parks the loop
	// inside its heartbeat branch, where it can't be selecting on the channel
	node.Mutex.Lock()
	time.Sleep(testLoopSettle)
	node.ResignLeader()
	node.Mutex.Unlock()

	assertLoopStopped(t, jobs)
	assert.Equal(t, Follower, node.GetRole())
}

func TestLeaderLoopKeepsItsOwnTickersAfterReelection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()

	// parks the first loop's goroutine on the mutex, so it can only see the
	// resign after the next term has replaced the tickers
	node.Mutex.Lock()
	time.Sleep(testLoopSettle)
	node.ResignLeader()
	node.becomeLeader(ctx)
	node.Mutex.Unlock()

	time.Sleep(testLoopSettle)
	before := jobs.Load()
	require.Eventually(t, func() bool { return jobs.Load() > before }, 5*time.Second, testLeaderInterval,
		"old loop stopped the new term's ticker")

	node.Mutex.Lock()
	node.ResignLeader()
	node.Mutex.Unlock()

	assertLoopStopped(t, jobs)
}

func TestLeaderLoopFromEarlierTermStopsOnNewTerm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)

	node.Mutex.Lock()
	node.becomeLeader(ctx)
	firstResign := node.Resign
	node.becomeLeader(ctx) // no resign in between
	node.Mutex.Unlock()

	select {
	case <-firstResign:
	case <-time.After(5 * time.Second):
		t.Fatal("earlier term's loop was not told to stop")
	}

	node.Mutex.Lock()
	node.ResignLeader()
	node.Mutex.Unlock()

	assertLoopStopped(t, jobs)
}
