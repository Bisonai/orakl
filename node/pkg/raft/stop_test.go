package raft

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Run returns only once a leader job in flight has, and no job starts after:
// a stopped leader publishes no further round.
func TestStoppedLeaderRunsNoJobAfterRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, jobs := newLeaderLoopNode(ctx, t)
	node.HandleCustomMessage = func(context.Context, Message) error { return nil }
	inJob := make(chan struct{}, 1)
	release := make(chan struct{})
	var finished atomic.Int64
	countJob := node.LeaderJob
	node.LeaderJob = func(ctx context.Context) error {
		select {
		case inJob <- struct{}{}:
			<-release // the first job blocks until the stop is under way
		default:
		}
		defer finished.Add(1)
		return countJob(ctx)
	}

	runCtx, stop := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		node.Run(runCtx)
	}()
	require.Eventually(t, func() bool { return node.GetRole() == Leader }, testElectionWait, 10*time.Millisecond)
	<-inJob

	stop()
	select {
	case <-done:
		t.Fatal("Run returned with a leader job in flight")
	case <-time.After(testLoopSettle):
	}
	close(release)
	<-done

	assert.Equal(t, Follower, node.GetRole())
	ran := jobs.Load()
	assert.Equal(t, ran, finished.Load(), "a job still running after Run returned")
	time.Sleep(testLoopSettle)
	assert.Equal(t, ran, jobs.Load(), "a job ran after Run returned")
}

// A granted vote handled after Run returned doesn't make the node lead.
func TestStoppedCandidateDoesNotLead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, jobs := newCandidate(ctx, t)
	cancel()
	require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))
	assert.Equal(t, Candidate, node.GetRole())
	time.Sleep(testLoopSettle)
	assert.Zero(t, jobs.Load())
}

// A wait that gave up doesn't stop the count from being used again (a stop
// that timed out, then a restarted Run leading).
func TestGoroutineCountReusedAfterAbandonedWait(t *testing.T) {
	var count goroutineCount
	select {
	case <-count.idleChan():
	default:
		t.Fatal("not idle before any add")
	}

	count.add()
	abandoned := count.idleChan()
	select {
	case <-abandoned:
		t.Fatal("idle with a goroutine running")
	case <-time.After(testLeaderInterval):
	}
	count.add() // while the abandoned wait is pending
	count.done()
	count.done()
	<-abandoned

	count.add()
	again := count.idleChan()
	count.done()
	<-again
}
