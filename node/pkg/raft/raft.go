package raft

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

const HEARTBEAT_TIMEOUT = 100 * time.Millisecond

// leaderClockEpoch gives leaderClock monotonic timestamps that fit in an atomic.
var leaderClockEpoch = time.Now()

func NewRaftNode(
	h host.Host,
	ps *pubsub.PubSub,
	topic *pubsub.Topic,
	messageBuffer int,
	leaderJobTimeout time.Duration,
) *Raft {
	r := &Raft{
		Host:  h,
		Ps:    ps,
		Topic: topic,

		Role:          "follower",
		VotedFor:      "",
		LeaderID:      "",
		VotesReceived: 0,
		Term:          0,
		Mutex:         sync.Mutex{},

		MessageBuffer:    make(chan *pubsub.Message, messageBuffer),
		Resign:           make(chan interface{}),
		HeartbeatTimeout: HEARTBEAT_TIMEOUT,

		LeaderJobTimeout: leaderJobTimeout,
		MissedHeartbeats: 0,
		CooldownPeriod:   DefaultCooldownPeriod,
		LastElectionTime: time.Time{},

		PeerSilenceTimeout: DefaultPeerSilenceTimeout,
		LeaderJobWatchdog:  DefaultLeaderJobWatchdog,

		SparseHeartbeat: SparseHeartbeatEnabled(),
	}
	return r
}

// Run blocks until ctx is done and the topic subscription is released. The
// topic itself stays open: its owner closes it, so a stopped node can Run again.
func (r *Raft) Run(ctx context.Context) {
	subDone := make(chan struct{})
	go func() {
		defer close(subDone)
		r.subscribe(ctx)
	}()
	defer func() { <-subDone }()
	if r.peerLeaveGrace() >= 0 {
		if events, err := r.Topic.EventHandler(); err != nil {
			log.Warn().Err(err).Str("Player", "Raft").Msg("no topic peer events, not electing on a leader leaving")
		} else {
			leavesDone := make(chan struct{})
			go func() {
				defer close(leavesDone)
				r.watchPeerLeaves(ctx, events)
			}()
			defer func() { <-leavesDone }()
		}
	}
	// a stopped node can't lead: without resigning, a restart keeps the Leader
	// role with no leader loop, so it never campaigns and refuses votes. Run
	// returns once the leader loop and its jobs have, so a stopped node
	// publishes no further round (becomeLeader refuses once ctx is done).
	defer func() {
		r.Mutex.Lock()
		if r.Role == Leader {
			r.ResignLeader()
		}
		r.Mutex.Unlock()
		r.waitLeaderLoops()
	}()
	r.Mutex.Lock()
	r.startElectionTimer()
	r.Mutex.Unlock()
	for {
		select {
		case rawMsg := <-r.MessageBuffer:
			// the signed author, not the payload's SentFrom
			r.stampPeer(rawMsg.GetFrom())
			go func(*pubsub.Message) {
				msg, err := r.unmarshalMessage(rawMsg.Data)
				if err != nil {
					log.Error().Err(err).Msg("failed to unmarshal message")
					return
				}

				err = r.handleFrom(ctx, msg, rawMsg.GetFrom().String())
				if err != nil {
					if errors.Is(err, errorSentinel.ErrAggregatorNonLeaderRaftMessage) {
						log.Debug().Err(err).Str("Player", "Raft").Msg("failed to handle message")
					} else {
						log.Error().Err(err).Str("Player", "Raft").Msg("failed to handle message")
					}
				}
			}(rawMsg)
		case <-r.ElectionTimer.C:
			r.startElection(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Raft) subscribe(ctx context.Context) {
	sub, err := r.Topic.Subscribe()
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to topic")
		return
	}
	// cancel only: closing the topic here made a restarted node unusable (#2783)
	defer sub.Cancel()
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("context cancelled")
			return
		default:
			rawMsg, err := sub.Next(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to get message from topic")
				continue
			}
			select {
			case r.MessageBuffer <- rawMsg:
			case <-ctx.Done():
				return
			}
		}
	}
}

// handler for incoming messages

// handleMessage handles a message whose payload SentFrom is taken as its author.
func (r *Raft) handleMessage(ctx context.Context, msg Message) error {
	return r.handleFrom(ctx, msg, msg.SentFrom)
}

// handleFrom handles a message published by author, the signed pubsub author.
// Only a message whose SentFrom is its author counts as the leader being heard
// (silence metrics, stickiness, liveness): another peer can't keep a leader
// alive by naming it (#2782).
func (r *Raft) handleFrom(ctx context.Context, msg Message, author string) error {
	authentic := msg.SentFrom == author
	// heartbeats record in handleHeartbeat, once validated
	if msg.Type != Heartbeat && authentic {
		r.recordLeaderSilence(author, r.LeaderMessageTypes[msg.Type])
	}
	if authentic {
		r.stampRivalLeader(msg.Type, author)
	}
	switch msg.Type {
	case Heartbeat:
		return r.handleHeartbeat(msg, authentic)
	case RequestVote:
		return r.handleRequestVote(ctx, msg)
	case ReplyVote:
		return r.handleReplyVote(ctx, msg)
	default:
		return r.HandleCustomMessage(ctx, msg)
	}
}

func (r *Raft) handleHeartbeat(msg Message, authentic bool) error {
	if msg.SentFrom == r.GetHostId() {
		return nil
	}
	var heartbeatMessage HeartbeatMessage
	err := decodeInner(msg.Data, &heartbeatMessage)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal heartbeat message")
		return err
	}

	if heartbeatMessage.LeaderID != msg.SentFrom {
		return errorSentinel.ErrRaftLeaderIdMismatch
	}

	// after validating: an invalid heartbeat must not split a real gap in two
	if authentic {
		r.recordLeaderSilence(msg.SentFrom, true)
	}

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.MissedHeartbeats = 0

	currentRole := r.Role
	currentTerm := r.Term

	// A node that campaigned past a live leader its peers still hear stays at the
	// higher term: the leader and those peers ignore its RequestVote. Without a
	// live leader of its own it follows this one, or it would ignore the leader's
	// heartbeats and keep campaigning against it for as long as it leads. A
	// leader at a higher term than the one it hears takes over, or a stale one
	// heard first would hold it off.
	followLower := heartbeatMessage.Term < currentTerm && currentRole != Leader &&
		(!r.hearsLeaderOtherThan(heartbeatMessage.LeaderID) ||
			heartbeatMessage.Term > r.leaderClock.Load().term)

	// ends a candidacy, or a late granted vote would still make this node leader
	if r.Role == Candidate && (heartbeatMessage.Term >= currentTerm || followLower) {
		r.Role = Follower
	}

	if r.Role == Follower {
		r.startElectionTimer()
	}

	if heartbeatMessage.Term > currentTerm {
		if currentRole == Leader {
			r.ResignLeader()
		}
		r.Term = heartbeatMessage.Term
		r.followLeader(heartbeatMessage.LeaderID, heartbeatMessage.Term)

		return nil
	} else if heartbeatMessage.Term == currentTerm {
		if currentRole == Leader {
			if r.GetHostId() < heartbeatMessage.LeaderID {
				r.ResignLeader()
				r.followLeader(heartbeatMessage.LeaderID, heartbeatMessage.Term)
			} else {
				return nil
			}
		} else {
			r.followLeader(heartbeatMessage.LeaderID, heartbeatMessage.Term)
		}
	} else if followLower {
		// it gave up campaigning, so its own vote at this term is void and
		// mustn't deny a candidate at this term once the leader is gone
		if r.VotedFor == r.GetHostId() {
			r.VotedFor = ""
		}
		r.followLeader(heartbeatMessage.LeaderID, heartbeatMessage.Term)
	}

	return nil
}

func (r *Raft) handleRequestVote(ctx context.Context, msg Message) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	// leader stickiness: neither a leader nor a node that still hears one grants
	// a vote or adopts the candidate's term, so a node that stalled briefly can't
	// depose a leader its peers hear. A follower still denies, so the candidate
	// hears from it and doesn't count it out as silent (liveVoters).
	if r.Role == Leader {
		return nil
	}
	if r.hearsLeaderOtherThan(msg.SentFrom) {
		return r.sendReplyVote(ctx, msg.SentFrom, false)
	}

	var requestVoteMessage RequestVoteMessage
	if err := decodeInner(msg.Data, &requestVoteMessage); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal request vote message")
		return err
	}

	if requestVoteMessage.Term < r.Term {
		return r.sendReplyVote(ctx, msg.SentFrom, false)
	}

	if requestVoteMessage.Term > r.Term {
		r.Term = requestVoteMessage.Term
		r.Role = Follower
		r.VotedFor = ""
	}

	if r.Role == Candidate && requestVoteMessage.Term == r.Term && msg.SentFrom != r.GetHostId() {
		r.Role = Follower
		return r.sendReplyVote(ctx, msg.SentFrom, false)
	}

	if r.VotedFor != "" && r.VotedFor != msg.SentFrom {
		return r.sendReplyVote(ctx, msg.SentFrom, false)
	}

	r.VotedFor = msg.SentFrom
	r.votedAt = time.Now()
	r.startElectionTimer()
	log.Debug().Bool("vote granted", true).Msg("voted")
	return r.sendReplyVote(ctx, msg.SentFrom, true)
}

func (r *Raft) handleReplyVote(ctx context.Context, msg Message) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.Role != Candidate {
		return nil
	}

	var replyVoteMessage ReplyRequestVoteMessage
	err := decodeInner(msg.Data, &replyVoteMessage)
	if err != nil {
		return err
	}

	if replyVoteMessage.LeaderID != r.GetHostId() {
		return nil
	}

	if replyVoteMessage.VoteGranted && replyVoteMessage.LeaderID == r.GetHostId() && r.Role == Candidate {
		r.VotesReceived++
		log.Debug().Int("vote received", r.VotesReceived).Msg("vote received")
		subscribed, live := r.voterCounts()
		if r.VotesReceived >= votesNeeded(subscribed) {
			r.becomeLeader(ctx)
		} else if r.VotesReceived >= votesNeeded(live) {
			r.recountAfterGrace(ctx)
		}
	}
	return nil
}

// hearsLeaderOtherThan reports whether a heartbeat or LeaderMessageTypes message
// from the current leader arrived within the minimum election timeout, and that
// leader isn't candidate. A leader campaigning has lost its role, and a node
// always votes for itself. Rounds' leader messages count, not only heartbeats,
// so this still holds once heartbeats are sent less often (#2747). Messages any
// node sends (votes, price data, proofs) don't: a deposed leader still sends
// them. A leader that goes quiet turns this false before any peer can time out;
// one whose jobs stop resigns (LeaderJobWatchdog), so it can't hold off an
// election that is needed for long.
func (r *Raft) hearsLeaderOtherThan(candidate string) bool {
	if candidate == r.GetHostId() {
		return false
	}
	clock := r.leaderClock.Load()
	if clock == nil || clock.id == candidate {
		return false
	}
	return since(&clock.activeAt) < r.minElectionTimeout()
}

// stampPeer records a message from a peer, for liveVoters. Runs for every
// received message, so lock-free and allocating only for a new peer.
func (r *Raft) stampPeer(id peer.ID) {
	if id == r.Host.ID() {
		return
	}
	now := int64(time.Since(leaderClockEpoch))
	if stamp, ok := r.peerSeen.Load(id); ok {
		storeLater(stamp.(*atomic.Int64), now)
		return
	}
	fresh := &atomic.Int64{}
	fresh.Store(now)
	if stamp, loaded := r.peerSeen.LoadOrStore(id, fresh); loaded {
		storeLater(stamp.(*atomic.Int64), now)
	}
}

// voterCounts returns the peers subscribed to the topic, and those of them that
// sent any message within PeerSilenceTimeout. A peer that hangs while still connected then stops
// counting, so a lone candidate wins on its own vote after the timeout instead
// of waiting for the connection to drop. A peer not heard from yet counts from
// when it's first seen subscribed. One that left is forgotten only once long
// silent too, so a hung peer whose connection flaps can't renew its grace. The
// caller holds Mutex.
func (r *Raft) voterCounts() (subscribed, live int) {
	timeout := r.PeerSilenceTimeout
	if timeout <= 0 {
		timeout = DefaultPeerSilenceTimeout // zero would count every peer out
	}
	listed := r.Ps.ListPeers(r.Topic.String())
	listedSet := make(map[peer.ID]bool, len(listed))
	for _, id := range listed {
		listedSet[id] = true
		stamp, ok := r.peerSeen.Load(id)
		if !ok {
			r.stampPeer(id)
			live++
			continue
		}
		if since(stamp.(*atomic.Int64)) < timeout {
			live++
		}
	}
	r.peerSeen.Range(func(id, stamp any) bool {
		if !listedSet[id.(peer.ID)] && since(stamp.(*atomic.Int64)) > 6*timeout {
			r.peerSeen.Delete(id)
		}
		return true
	})
	return len(listed), live
}

// recountAfterGrace wins the election a minimum election timeout from now if
// the votes are then still enough, counting only live voters. A win that relies
// on counting a silent peer out waits this long: followers only talk during
// rounds, so a healthy peer can look silent until it answers this election, and
// two candidates that each count the other out would both lead. A peer that's
// alive answers within the grace and counts again. The caller holds Mutex.
func (r *Raft) recountAfterGrace(ctx context.Context) {
	if r.recountTerm == r.Term {
		return
	}
	term := r.Term
	r.recountTerm = term
	time.AfterFunc(r.minElectionTimeout(), func() {
		r.Mutex.Lock()
		defer r.Mutex.Unlock()
		if ctx.Err() != nil || r.Role != Candidate || r.Term != term {
			return
		}
		if _, live := r.voterCounts(); r.VotesReceived >= votesNeeded(live) {
			r.becomeLeader(ctx)
		}
	})
}

// votesNeeded is a strict majority of the voters: the live peers on this node's
// topic (liveVoters) plus the node itself, whose own vote is counted. A lone node wins
// on its own vote, so it keeps serving while every peer is gone; with any peer it
// needs that peer's grant too.
func votesNeeded(peers int) int {
	return (peers+1)/2 + 1
}

// publishing messages

func (r *Raft) PublishMessage(ctx context.Context, msg Message) error {
	msg.Timestamp = time.Now()
	data, err := encodeMessage(msg)
	if err != nil {
		return err
	}
	if err = r.Topic.Publish(ctx, data); err != nil {
		return err
	}
	metrics.RecordP2PPublished(string(msg.Type), len(data))
	r.stampLeaderMessageSent(msg.Type, msg.Timestamp)
	return nil
}

// sendHeartbeat publishes for the leader loop that owns resign, unless that loop
// has lost its term. The check is under Mutex, where resign gets closed: a loop
// parked here during a resign would otherwise publish the term just adopted from
// the new leader.
func (r *Raft) sendHeartbeat(ctx context.Context, resign <-chan interface{}) error {
	r.Mutex.Lock()
	if resigned(resign) {
		r.Mutex.Unlock()
		return nil
	}
	heartbeatMessage := HeartbeatMessage{
		LeaderID: r.GetHostId(),
		Term:     r.Term,
	}
	r.Mutex.Unlock()
	marshalledHeartbeatMsg, err := encodeInner(heartbeatMessage)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal heartbeat message")
		return err
	}

	message := Message{
		Type:     Heartbeat,
		SentFrom: r.GetHostId(),
		Data:     marshalledHeartbeatMsg,
	}
	sentAt := time.Now()
	err = r.PublishMessage(ctx, message)
	if err != nil {
		log.Error().Err(err).Msg("failed to send heartbeat")
		return err
	}
	r.stampHeartbeatSent(resign, sentAt)
	return nil
}

func (r *Raft) sendReplyVote(ctx context.Context, to string, voteGranted bool) error {
	replyVoteMessage := ReplyRequestVoteMessage{
		VoteGranted: voteGranted,
		LeaderID:    to,
	}
	marshalledReplyVoteMsg, err := encodeInner(replyVoteMessage)
	if err != nil {
		return err
	}
	message := Message{
		Type:     ReplyVote,
		SentFrom: r.GetHostId(),
		Data:     marshalledReplyVoteMsg,
	}
	err = r.PublishMessage(ctx, message)
	if err != nil {
		return err
	}
	return nil
}

func (r *Raft) sendRequestVote(ctx context.Context) error {
	requestVoteMessage := RequestVoteMessage{
		Term: r.Term,
	}
	marshalledRequestVoteMsg, err := encodeInner(requestVoteMessage)
	if err != nil {
		return err
	}

	message := Message{
		Type:     RequestVote,
		SentFrom: r.GetHostId(),
		Data:     marshalledRequestVoteMsg,
	}
	err = r.PublishMessage(ctx, message)
	if err != nil {
		return err
	}
	return nil
}

// utility functions

func resigned(resign <-chan interface{}) bool {
	select {
	case <-resign:
		return true
	default:
		return false
	}
}

func (r *Raft) ResignLeader() {
	if r.Resign != nil {
		close(r.Resign)
		r.Resign = nil
		r.Role = Follower
		r.LeaderID = ""
		r.leaderClock.Store(nil)
		r.startElectionTimer()
	}
}

// followLeader sets another peer as the current leader at the term it leads in,
// and restarts the silence clock from now only if it changed. The term can be
// below this node's own (see handleHeartbeat). The caller holds Mutex.
func (r *Raft) followLeader(leaderID string, term int) {
	if r.LeaderID == leaderID {
		if clock := r.leaderClock.Load(); clock != nil {
			clock.term = term // re-elected
		}
		return
	}
	r.LeaderID = leaderID
	now := int64(time.Since(leaderClockEpoch))
	clock := &leaderClock{id: leaderID, term: term, followedAt: now}
	clock.lastSeen.Store(now)
	clock.activeAt.Store(now)
	r.recordLeaderChangeSilence(r.leaderClock.Swap(clock), term)
}

// recordLeaderChangeSilence observes how long the outgoing leader had been silent
// when it was replaced, which recordLeaderSilence can't see: no further message
// matches its clock. old is nil when there was no leader to replace. Two live
// leaders at one term flip LeaderID on each heartbeat, which isn't a failover, so
// it's skipped. Same term alone isn't enough, as terms aren't unique across nodes:
// a real failover also shows silence past the missed heartbeats and minimum election
// timers that detecting it takes (see getRandomElectionTimeout, startElection).
// term is the new leader's. The caller holds Mutex.
func (r *Raft) recordLeaderChangeSilence(old *leaderClock, term int) {
	if old == nil {
		return
	}
	gap := since(&old.lastSeen)
	if old.term == term && gap < (MaxMissedHeartbeats+1)*5*r.HeartbeatTimeout {
		return
	}
	metrics.RecordRaftLeaderChangeSilence(gap)
}

// recordLeaderSilence observes the gap since the previous message from the
// current leader. The message that makes a leader runs before its clock exists,
// so it records nothing and no gap spans two leaders. Own messages never match:
// the clock is nil while this node leads, and otherwise holds another peer. Runs
// for every received message, so lock-free and no allocation. A message other
// than a vote also stamps activeAt, for leader stickiness.
func (r *Raft) recordLeaderSilence(sentFrom string, active bool) {
	clock := r.leaderClock.Load()
	if clock == nil || clock.id != sentFrom {
		return
	}
	now := int64(time.Since(leaderClockEpoch))
	if active {
		storeLater(&clock.activeAt, now)
	}
	// an older message landing last would make the next gap too long
	if prev, moved := storeLater(&clock.lastSeen, now); moved {
		metrics.RecordRaftLeaderSilence(time.Duration(now - prev))
	}
}

// storeLater moves stamp forward to now, and returns the value it replaced.
// Handlers run concurrently, so an older message landing last must not move it
// back: then it reports false.
func storeLater(stamp *atomic.Int64, now int64) (prev int64, moved bool) {
	for {
		prev = stamp.Load()
		if now <= prev {
			return prev, false
		}
		if stamp.CompareAndSwap(prev, now) {
			return prev, true
		}
	}
}

// since returns the time elapsed from a leaderClock timestamp.
func since(stamp *atomic.Int64) time.Duration {
	// load first: the stamp can't then be ahead of now
	at := stamp.Load()
	return time.Since(leaderClockEpoch) - time.Duration(at)
}

func (r *Raft) setLeaderState() {
	// a loop from an earlier term that never resigned must not outlive this one
	if r.Resign != nil {
		close(r.Resign)
	}
	r.Resign = make(chan interface{})
	r.ElectionTimer.Stop()
	r.Term++
	r.Role = Leader
	r.LeaderID = r.GetHostId()
	r.recordLeaderChangeSilence(r.leaderClock.Swap(nil), r.Term)
	// no skipping until this term's first heartbeat publishes (skipHeartbeat)
	if r.SparseHeartbeat {
		r.heartbeatSentAt.Store(neverSent)
	}
}

// becomeLeader takes the role and starts the leader loop, unless ctx is done:
// Run has returned or is returning, and a vote reply still being handled must
// not make a stopped node lead. The caller holds Mutex.
func (r *Raft) becomeLeader(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	r.setLeaderState()

	// captured before the goroutine starts: ResignLeader nils r.Resign and a
	// later term replaces it, so the loop only uses its own
	resign := r.Resign
	heartbeatTicker := time.NewTicker(r.HeartbeatTimeout)
	leaderJobTicker := time.NewTicker(r.LeaderJobTimeout)
	// a job interval near the watchdog would trip it between healthy jobs
	watchdog := max(r.LeaderJobWatchdog, 3*r.LeaderJobTimeout)
	if r.LeaderJobWatchdog <= 0 {
		watchdog = max(DefaultLeaderJobWatchdog, 3*r.LeaderJobTimeout)
	}
	// last successful job of this term, from when it began
	jobDone := &atomic.Int64{}
	jobDone.Store(int64(time.Since(leaderClockEpoch)))
	runJob := func() {
		if ctx.Err() != nil {
			return
		}
		if err := r.LeaderJob(ctx); err != nil {
			log.Error().Err(err).Msg("failed to execute leader job")
			return
		}
		storeLater(jobDone, int64(time.Since(leaderClockEpoch)))
	}

	r.leaderLoops.add()
	go func() {
		defer r.leaderLoops.done()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("recovered from panic in leader job: %v", r)
			}
		}()
		defer heartbeatTicker.Stop()
		defer leaderJobTicker.Stop()

		err := r.sendHeartbeat(ctx, resign)
		if err != nil {
			log.Error().Err(err).Msg("failed to send heartbeat")
		}
		if resigned(resign) {
			return
		}
		runJob()

		for {
			// select below picks at random, so a tick queued during the last
			// iteration could otherwise run after a resign
			if resigned(resign) {
				return
			}

			select {
			case <-resign:
				return

			case <-heartbeatTicker.C:
				if since(jobDone) > watchdog {
					if r.resignStuckLeader(resign, watchdog) {
						return
					}
					// no peer to take over: check again a watchdog from now
					storeLater(jobDone, int64(time.Since(leaderClockEpoch)))
				}
				if r.skipHeartbeat() {
					metrics.RecordRaftHeartbeatSkipped()
					continue
				}
				err := r.sendHeartbeat(ctx, resign)
				if err != nil {
					log.Error().Err(err).Msg("failed to send heartbeat")
				}

			case <-leaderJobTicker.C:
				r.leaderLoops.add()
				go func() {
					defer r.leaderLoops.done()
					defer func() {
						if r := recover(); r != nil {
							log.Error().Msgf("recovered from panic in LeaderJob: %v", r)
						}
					}()
					// a resign can land after this tick was picked
					if resigned(resign) {
						return
					}
					runJob()
				}()

			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				return
			}
		}
	}()
}

// leaderLoopStopWait bounds how long a stopping Run waits for the leader loop
// and its jobs, so a job stuck in a publish can't hold a stop forever.
const leaderLoopStopWait = 5 * time.Second

// waitLeaderLoops waits for every leader loop and leader job goroutine to
// return, up to leaderLoopStopWait. Called without Mutex: they take it.
func (r *Raft) waitLeaderLoops() {
	select {
	case <-r.leaderLoops.idleChan():
	case <-time.After(leaderLoopStopWait):
		log.Warn().Str("Player", "Raft").Dur("waited", leaderLoopStopWait).Msg("leader job still running after stop")
	}
}

func (r *Raft) minElectionTimeout() time.Duration {
	return r.HeartbeatTimeout * 5
}

// resignStuckLeader gives up a term whose leader jobs stopped succeeding, while
// its heartbeats would keep peers following it: rounds stop but no election
// happens (e.g. trigger publishes failing). A stall on Mutex (#2749) stops the
// heartbeats too, so peers time out and count it out (liveVoters) instead.
//
// It stops the heartbeats, and holds off campaigning for twice the longest a
// peer takes to notice and campaign, so a peer takes over rather than this node
// winning the term back. With no live peer to take over it keeps the term, as
// resigning would only add an outage. Reports whether the loop must stop.
func (r *Raft) resignStuckLeader(resign <-chan interface{}, watchdog time.Duration) bool {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if resigned(resign) {
		return true
	}
	if _, live := r.voterCounts(); live == 0 {
		return false
	}
	log.Warn().Str("Player", "Raft").Dur("watchdog", watchdog).Msg("no leader job succeeded, resigning")
	r.ResignLeader()
	r.MissedHeartbeats = 0
	r.campaignAfter = time.Now().Add(2 * r.maxElectionDetection())
	return true
}

// maxElectionDetection is the longest a follower takes from a leader's last
// heartbeat to campaigning: every missed heartbeat plus the one that starts the
// election, each up to the maximum election timeout.
func (r *Raft) maxElectionDetection() time.Duration {
	return (MaxMissedHeartbeats + 1) * 2 * r.minElectionTimeout()
}

// heardLeaderSinceTimerArmed reports whether a LeaderMessageTypes message (or a
// heartbeat) from the current leader arrived after the election timer was last
// armed. It reads the leader's lock-free activeAt stamp, so the message path
// takes no Mutex for it (#2749). Leader messages carry no term: only the leader
// this node follows counts, and they never change who that is, which only a
// heartbeat (with its term) or an election does. A deposed leader's late
// trigger therefore counts only on a follower that hasn't heard of its
// successor yet, and holds off its election by at most one timer period, as a
// late heartbeat would. The caller holds Mutex.
func (r *Raft) heardLeaderSinceTimerArmed() bool {
	clock := r.leaderClock.Load()
	if clock == nil {
		return false
	}
	// the heartbeat that made it the leader already reset the timer
	active := clock.activeAt.Load()
	return active > r.timerArmedAt && active > clock.followedAt
}

func (r *Raft) getRandomElectionTimeout() time.Duration {
	baseTimeout := r.minElectionTimeout()
	jitter := time.Duration(rand.Int63n(int64(r.minElectionTimeout())))
	return baseTimeout + jitter
}

// startElectionTimer (re)arms the election timer. The caller holds Mutex, or
// nothing else runs yet.
func (r *Raft) startElectionTimer() {
	r.timerArmedAt = int64(time.Since(leaderClockEpoch))
	if r.ElectionTimer != nil {
		if !r.ElectionTimer.Stop() {
			select {
			case <-r.ElectionTimer.C:
				log.Debug().Msg("Old timer channel drained")
			default:
				log.Debug().Msg("Old timer channel already empty")
			}
		}
		r.ElectionTimer.Reset(r.getRandomElectionTimeout())
	} else {
		r.ElectionTimer = time.NewTimer(r.getRandomElectionTimeout())
	}
}

func (r *Raft) startElection(ctx context.Context) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	// the timer can fire just as a winning vote is handled, so this runs on the
	// new leader. Stays disarmed: ResignLeader, the only way out of the role,
	// restarts it.
	if r.Role == Leader {
		return
	}

	// a trigger or price fix from the leader since the timer was armed counts
	// as a heartbeat, so a leader busy with rounds isn't timed out
	if r.Role == Follower && r.heardLeaderSinceTimerArmed() {
		r.MissedHeartbeats = 0
		r.startElectionTimer()
		return
	}

	if r.MissedHeartbeats < MaxMissedHeartbeats {
		r.MissedHeartbeats++
		log.Debug().Int("missed heartbeats", r.MissedHeartbeats).Msg("missed heartbeats")
		r.startElectionTimer()
		return
	}

	r.campaign(ctx, metrics.RaftElectionReasonTimeout)
}

// campaign starts an election for reason, unless a stuck term's hold-off or the
// cooldown is active: then the timer retries it, with no missed heartbeats left
// to count. The caller holds Mutex.
func (r *Raft) campaign(ctx context.Context, reason string) {
	// a stopping node must not bump its term and solicit votes
	if ctx.Err() != nil {
		return
	}
	if time.Now().Before(r.campaignAfter) {
		log.Debug().Msg("resigned a stuck term, skipping election.")
		r.MissedHeartbeats = MaxMissedHeartbeats
		r.startElectionTimer()
		return
	}

	if !r.LastElectionTime.IsZero() && time.Since(r.LastElectionTime) < r.CooldownPeriod {
		log.Debug().Msg("Election cooldown period active, skipping election.")
		r.MissedHeartbeats = MaxMissedHeartbeats
		r.startElectionTimer()
		return
	}

	log.Debug().Str("reason", reason).Msg("start election")
	metrics.RecordRaftElectionStarted(reason)
	r.Term++
	r.VotesReceived = 0
	r.Role = Candidate
	r.VotedFor = r.GetHostId()
	r.MissedHeartbeats = 0

	r.LastElectionTime = time.Now()

	r.startElectionTimer()

	err := r.sendRequestVote(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to send request vote")
	}
}

func (r *Raft) unmarshalMessage(data []byte) (Message, error) {
	return decodeMessage(data)
}

// goroutineCount counts running goroutines, with a channel closed whenever the
// count is zero. Unlike a WaitGroup it can be added to while a wait that timed
// out is abandoned, so a stop that gave up can't break a later Run. The zero
// value is ready.
type goroutineCount struct {
	mu   sync.Mutex
	n    int
	idle chan struct{} // closed while n is 0; nil until the first add
}

func (c *goroutineCount) add() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n == 0 {
		c.idle = make(chan struct{})
	}
	c.n++
}

func (c *goroutineCount) done() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n--
	if c.n == 0 {
		close(c.idle)
	}
}

// idleChan returns a channel that is closed once the count is zero.
func (c *goroutineCount) idleChan() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n == 0 {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return c.idle
}
