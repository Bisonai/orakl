package raft

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// time for a cluster to elect: missed heartbeats, election timers, cooldowns
// and split votes, with room for a loaded CI runner
const testElectionWait = 30 * time.Second

func TestVotesNeeded(t *testing.T) {
	for peers, want := range map[int]int{0: 1, 1: 2, 2: 2, 3: 3, 4: 3, 5: 4} {
		assert.Equal(t, want, votesNeeded(peers), "%d peers", peers)
	}
}

func requestVoteFrom(t *testing.T, candidate string, term int) Message {
	t.Helper()
	data, err := encodeInner(RequestVoteMessage{Term: term})
	require.NoError(t, err)
	return Message{Type: RequestVote, SentFrom: candidate, Data: data}
}

func voteState(node *Raft) (term int, votedFor string, role RoleType) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	return node.Term, node.VotedFor, node.Role
}

func TestRequestVoteIgnoredWhileLeaderHeard(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newLeaderLoopNode(ctx, t)
	leader, candidate := testPeerID(t), testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 3)))

	require.NoError(t, node.handleMessage(ctx, requestVoteFrom(t, candidate, 4)))
	term, votedFor, role := voteState(node)
	assert.Equal(t, 3, term, "term adopted from a disruptive candidate")
	assert.Empty(t, votedFor)
	assert.Equal(t, Follower, role)

	// the leader is silent past the minimum election timeout
	time.Sleep(node.minElectionTimeout() + testLeaderInterval)
	require.NoError(t, node.handleMessage(ctx, requestVoteFrom(t, candidate, 4)))
	term, votedFor, _ = voteState(node)
	assert.Equal(t, 4, term)
	assert.Equal(t, candidate, votedFor)
}

// Any leader message holds off a vote, so stickiness survives sparser heartbeats
// (#2747). A vote from the leader doesn't: it is no sign of leading.
func TestRequestVoteStickinessCountsLeaderMessages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for name, tc := range map[string]struct {
		msgType MessageType
		granted bool
	}{
		"trigger":   {"trigger", false},
		"replyVote": {ReplyVote, true},
	} {
		t.Run(name, func(t *testing.T) {
			node, _ := newLeaderLoopNode(ctx, t)
			node.HandleCustomMessage = func(context.Context, Message) error { return nil }
			leader, candidate := testPeerID(t), testPeerID(t)
			require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 3)))

			time.Sleep(node.minElectionTimeout() + testLeaderInterval)
			data, err := encodeInner(ReplyRequestVoteMessage{VoteGranted: true, LeaderID: candidate})
			require.NoError(t, err)
			require.NoError(t, node.handleMessage(ctx, Message{Type: tc.msgType, SentFrom: leader, Data: data}))
			require.NoError(t, node.handleMessage(ctx, requestVoteFrom(t, candidate, 4)))

			term, votedFor, _ := voteState(node)
			if tc.granted {
				assert.Equal(t, 4, term)
				assert.Equal(t, candidate, votedFor)
			} else {
				assert.Equal(t, 3, term)
				assert.Empty(t, votedFor)
			}
		})
	}
}

// A leader that campaigns has lost its role, so hearing it is no reason to refuse.
func TestRequestVoteFromHeardLeaderGranted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newLeaderLoopNode(ctx, t)
	leader := testPeerID(t)
	require.NoError(t, node.handleMessage(ctx, heartbeatFrom(t, leader, 3)))

	require.NoError(t, node.handleMessage(ctx, requestVoteFrom(t, leader, 4)))
	term, votedFor, _ := voteState(node)
	assert.Equal(t, 4, term)
	assert.Equal(t, leader, votedFor)
}

func TestRequestVoteIgnoredByLeader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newCandidate(ctx, t)
	require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))
	term := node.GetCurrentTerm()

	require.NoError(t, node.handleMessage(ctx, requestVoteFrom(t, testPeerID(t), term+1)))
	assert.Equal(t, Leader, node.GetRole())
	assert.Equal(t, term, node.GetCurrentTerm())
}

// startCluster runs n fully connected nodes, each first passed to configure, and
// waits until each sees the others on the topic.
func startCluster(ctx context.Context, t *testing.T, n int, configure ...func(*Raft)) []*Raft {
	t.Helper()
	nodes := make([]*Raft, 0, n)
	for i := 0; i < n; i++ {
		h, err := libp2psetup.NewHost(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { _ = h.Close() })
		ps, err := libp2psetup.MakePubsub(ctx, h)
		require.NoError(t, err)
		for _, other := range nodes {
			require.NoError(t, h.Connect(ctx, peer.AddrInfo{ID: other.Host.ID(), Addrs: other.Host.Addrs()}))
		}
		topic, err := ps.Join(topicString)
		require.NoError(t, err)

		node := NewRaftNode(h, ps, topic, 100, time.Second)
		node.LeaderJob = func(context.Context) error { return nil }
		node.HandleCustomMessage = func(context.Context, Message) error { return nil }
		for _, c := range configure {
			c(node)
		}
		nodes = append(nodes, node)
	}
	for _, node := range nodes {
		go node.Run(ctx)
	}
	require.Eventually(t, func() bool {
		for _, node := range nodes {
			if node.SubscribersCount() != n-1 {
				return false
			}
		}
		return true
	}, testElectionWait, 50*time.Millisecond, "peers not subscribed")
	return nodes
}

// singleLeader returns the leader when exactly one node leads and every other
// node follows it. A follower can be at a higher term than its leader, when it
// campaigned while the leader was heard.
func singleLeader(nodes []*Raft) (*Raft, bool) {
	var leader *Raft
	for _, node := range nodes {
		if node.GetRole() == Leader {
			if leader != nil {
				return nil, false
			}
			leader = node
		}
	}
	if leader == nil {
		return nil, false
	}
	id := leader.GetHostId()
	for _, node := range nodes {
		if node == leader {
			continue
		}
		if node.GetRole() != Follower || node.GetLeader() != id {
			return nil, false
		}
	}
	return leader, true
}

func waitForSingleLeader(t *testing.T, nodes []*Raft, msg string) *Raft {
	t.Helper()
	var leader *Raft
	require.Eventually(t, func() bool {
		var ok bool
		leader, ok = singleLeader(nodes)
		return ok
	}, testElectionWait, 50*time.Millisecond, msg)
	return leader
}

func forceNextElection(node *Raft) {
	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	node.MissedHeartbeats = MaxMissedHeartbeats
	node.CooldownPeriod = 0
}

func otherNode(nodes []*Raft, node *Raft) *Raft {
	for _, n := range nodes {
		if n != node {
			return n
		}
	}
	return nil
}

func TestLoneCandidateWins(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 1)
	waitForSingleLeader(t, nodes, "lone node did not elect itself")
}

// The flap seen on the two mainnet nodes: a follower that stalls past its
// election timeout must not win on its own vote while its peer still leads.
func TestStalledFollowerDoesNotDeposeLeader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 2)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	follower := otherNode(nodes, leader)
	term := leader.GetCurrentTerm()

	forceNextElection(follower)
	follower.startElection(ctx)
	require.Greater(t, follower.GetCurrentTerm(), term, "no election started")

	deadline := time.Now().Add(10 * leader.minElectionTimeout())
	for time.Now().Before(deadline) {
		require.Equal(t, Leader, leader.GetRole(), "leader deposed")
		require.Equal(t, term, leader.GetCurrentTerm(), "leader adopted the candidate's term")
		require.NotEqual(t, Leader, follower.GetRole(), "candidate won without its peer's vote")
		time.Sleep(testLeaderInterval)
	}
	// back to following, at its higher term
	assert.Equal(t, Follower, follower.GetRole())
	assert.Equal(t, leader.GetHostId(), follower.GetLeader())
}

func TestSurvivorTakesOverWhenLeaderDies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 2)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	survivor := otherNode(nodes, leader)

	start := time.Now()
	require.NoError(t, leader.Host.Close())
	waitForSingleLeader(t, []*Raft{survivor}, "survivor did not take over")
	t.Logf("survivor led after %s", time.Since(start))
}

// A leader that is still connected but no longer heard is replaced by the peers
// that form a majority without it.
func TestSilentLeaderReplacedByMajority(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 3)
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	others := []*Raft{}
	for _, node := range nodes {
		if node != leader {
			others = append(others, node)
		}
	}

	// sendHeartbeat takes the mutex, so holding it silences the leader
	leader.Mutex.Lock()
	var unlock sync.Once
	release := func() { unlock.Do(leader.Mutex.Unlock) }
	defer release()

	waitForSingleLeader(t, others, "no leader elected without the silent one")
	release()
	newLeader := waitForSingleLeader(t, nodes, "old leader did not rejoin")
	assert.NotEqual(t, leader.GetHostId(), newLeader.GetHostId())
}

func TestSimultaneousCandidatesConverge(t *testing.T) {
	for _, n := range []int{2, 3} {
		t.Run(fmt.Sprintf("%d nodes", n), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			nodes := startCluster(ctx, t, n)
			leader := waitForSingleLeader(t, nodes, "no initial leader")
			term := leader.GetCurrentTerm()

			leader.Mutex.Lock()
			leader.ResignLeader()
			leader.Mutex.Unlock()

			// every node campaigns at once, at the same term
			start := make(chan struct{})
			var wg sync.WaitGroup
			for _, node := range nodes {
				forceNextElection(node)
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					node.startElection(ctx)
				}()
			}
			close(start)
			wg.Wait()

			newLeader := waitForSingleLeader(t, nodes, "candidates did not converge")
			assert.Greater(t, newLeader.GetCurrentTerm(), term)
		})
	}
}

func TestLiveVotersSkipsSilentPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := startCluster(ctx, t, 2)
	node, peerID := nodes[0], nodes[1].GetHostId()
	setSeen := func(ago time.Duration) {
		stamp, _ := node.peerSeen.LoadOrStore(peerID, &atomic.Int64{})
		stamp.(*atomic.Int64).Store(int64(time.Since(leaderClockEpoch) - ago))
	}

	node.Mutex.Lock()
	defer node.Mutex.Unlock()
	setSeen(node.PeerSilenceTimeout / 2) // a stall below the timeout
	assert.Equal(t, 1, node.liveVoters())
	setSeen(node.PeerSilenceTimeout + time.Second)
	assert.Equal(t, 0, node.liveVoters())

	// a subscribed peer not heard from yet counts from now
	node.peerSeen.Delete(peerID)
	assert.Equal(t, 1, node.liveVoters())
	_, ok := node.peerSeen.Load(peerID)
	assert.True(t, ok)
}

// A leader that hangs but stays connected (subscribed, sends nothing) is replaced
// on 2 nodes once it has been silent for PeerSilenceTimeout.
func TestHungConnectedLeaderReplaced(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const silence = 2 * time.Second
	nodes := startCluster(ctx, t, 2, func(r *Raft) { r.PeerSilenceTimeout = silence })
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	survivor := otherNode(nodes, leader)

	// sendHeartbeat takes the mutex, and the test LeaderJob publishes nothing
	leader.Mutex.Lock()
	var unlock sync.Once
	release := func() { unlock.Do(leader.Mutex.Unlock) }
	defer release()

	start := time.Now()
	require.Eventually(t, func() bool { return survivor.GetRole() == Leader }, testElectionWait, 50*time.Millisecond, "survivor did not take over")
	took := time.Since(start)
	t.Logf("survivor led after %s", took)
	// its last heartbeat came at most an interval before the lock
	assert.GreaterOrEqual(t, took, silence-2*survivor.HeartbeatTimeout, "won before the peer counted as silent")

	release()
	newLeader := waitForSingleLeader(t, nodes, "hung leader did not rejoin")
	assert.Equal(t, survivor, newLeader)
}

// Heartbeats alive, rounds dead: the leader resigns after LeaderJobWatchdog and
// its peer takes over, not the resigned node again.
func TestStuckLeaderJobResigns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const watchdog = time.Second
	failing := map[*Raft]*atomic.Bool{}
	nodes := startCluster(ctx, t, 2, func(r *Raft) {
		r.LeaderJobWatchdog = watchdog
		r.LeaderJobTimeout = 100 * time.Millisecond
		fail := &atomic.Bool{}
		failing[r] = fail
		r.LeaderJob = func(context.Context) error {
			if fail.Load() {
				return fmt.Errorf("stuck")
			}
			return nil
		}
	})
	leader := waitForSingleLeader(t, nodes, "no initial leader")
	peer := otherNode(nodes, leader)
	term := leader.GetCurrentTerm()

	// healthy jobs keep it leading past the watchdog
	time.Sleep(2 * watchdog)
	require.Equal(t, Leader, leader.GetRole())
	require.Equal(t, term, leader.GetCurrentTerm())

	failing[leader].Store(true)
	require.Eventually(t, func() bool { return leader.GetRole() != Leader }, 3*watchdog, 20*time.Millisecond, "stuck leader did not resign")

	newLeader := waitForSingleLeader(t, nodes, "peer did not take over")
	assert.Equal(t, peer, newLeader, "resigned node won its term back")
}

func TestResignedStuckLeaderHoldsOffCampaign(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, _ := newCandidate(ctx, t)
	require.NoError(t, node.handleMessage(ctx, grantedVoteFor(t, node)))
	require.Equal(t, Leader, node.GetRole())
	node.Mutex.Lock()
	resign := node.Resign
	node.Mutex.Unlock()

	node.resignStuckLeader(resign, node.LeaderJobWatchdog)
	require.Equal(t, Follower, node.GetRole())
	term := node.GetCurrentTerm()

	forceNextElection(node)
	node.startElection(ctx)
	assert.Equal(t, Follower, node.GetRole())
	assert.Equal(t, term, node.GetCurrentTerm())
}
