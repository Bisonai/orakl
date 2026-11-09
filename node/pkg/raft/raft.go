package raft

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	errorSentinel "bisonai.com/miko/node/pkg/error"
	"bisonai.com/miko/node/pkg/metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
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
	}
	return r
}

func (r *Raft) Run(ctx context.Context) {
	go r.subscribe(ctx)
	r.startElectionTimer()
	for {
		select {
		case rawMsg := <-r.MessageBuffer:
			go func(*pubsub.Message) {
				msg, err := r.unmarshalMessage(rawMsg.Data)
				if err != nil {
					log.Error().Err(err).Msg("failed to unmarshal message")
					return
				}

				err = r.handleMessage(ctx, msg)
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
	defer func() {
		sub.Cancel()
		r.Topic.Close()
	}()
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
			r.MessageBuffer <- rawMsg
		}
	}
}

// handler for incoming messages

func (r *Raft) handleMessage(ctx context.Context, msg Message) error {
	// heartbeats record in handleHeartbeat, once validated
	if msg.Type != Heartbeat {
		r.recordLeaderSilence(msg.SentFrom)
	}
	switch msg.Type {
	case Heartbeat:
		return r.handleHeartbeat(msg)
	case RequestVote:
		return r.handleRequestVote(ctx, msg)
	case ReplyVote:
		return r.handleReplyVote(ctx, msg)
	default:
		return r.HandleCustomMessage(ctx, msg)
	}
}

func (r *Raft) handleHeartbeat(msg Message) error {
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
	r.recordLeaderSilence(msg.SentFrom)

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	r.MissedHeartbeats = 0

	currentRole := r.Role
	currentTerm := r.Term

	// ends a candidacy, or a late granted vote would still make this node leader
	if r.Role == Candidate && heartbeatMessage.Term >= currentTerm {
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
		r.followLeader(heartbeatMessage.LeaderID)

		return nil
	} else if heartbeatMessage.Term == currentTerm {
		if currentRole == Leader {
			if r.GetHostId() < heartbeatMessage.LeaderID {
				r.ResignLeader()
				r.followLeader(heartbeatMessage.LeaderID)
			} else {
				return nil
			}
		} else {
			r.followLeader(heartbeatMessage.LeaderID)
		}
	}

	return nil
}

func (r *Raft) handleRequestVote(ctx context.Context, msg Message) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.Role == Leader {
		return nil
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
		log.Debug().Int("subscribers count", r.SubscribersCount()).Msg("subscribers count")
		if r.VotesReceived >= (r.SubscribersCount()+1)/2 {
			r.becomeLeader(ctx)
		}
	}
	return nil
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
	err = r.PublishMessage(ctx, message)
	if err != nil {
		log.Error().Err(err).Msg("failed to send heartbeat")
		return err
	}
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

// followLeader sets another peer as the current leader, and restarts the silence
// clock from now only if it changed. The caller holds Mutex.
func (r *Raft) followLeader(leaderID string) {
	if r.LeaderID == leaderID {
		if clock := r.leaderClock.Load(); clock != nil {
			clock.term = r.Term // re-elected
		}
		return
	}
	r.LeaderID = leaderID
	clock := &leaderClock{id: leaderID, term: r.Term}
	clock.lastSeen.Store(int64(time.Since(leaderClockEpoch)))
	r.recordLeaderChangeSilence(r.leaderClock.Swap(clock))
}

// recordLeaderChangeSilence observes how long the outgoing leader had been silent
// when it was replaced, which recordLeaderSilence can't see: no further message
// matches its clock. old is nil when there was no leader to replace. Two live
// leaders at one term flip LeaderID on each heartbeat, which isn't a failover, so
// it's skipped. Same term alone isn't enough, as terms aren't unique across nodes:
// a real failover also shows silence past the missed heartbeats and minimum election
// timers that detecting it takes (see getRandomElectionTimeout, startElection). The
// caller holds Mutex.
func (r *Raft) recordLeaderChangeSilence(old *leaderClock) {
	if old == nil {
		return
	}
	// load first: lastSeen can't then be ahead of now
	lastSeen := old.lastSeen.Load()
	gap := time.Since(leaderClockEpoch) - time.Duration(lastSeen)
	if old.term == r.Term && gap < (MaxMissedHeartbeats+1)*5*r.HeartbeatTimeout {
		return
	}
	metrics.RecordRaftLeaderChangeSilence(gap)
}

// recordLeaderSilence observes the gap since the previous message from the
// current leader. The message that makes a leader runs before its clock exists,
// so it records nothing and no gap spans two leaders. Own messages never match:
// the clock is nil while this node leads, and otherwise holds another peer. Runs
// for every received message, so lock-free and no allocation.
func (r *Raft) recordLeaderSilence(sentFrom string) {
	clock := r.leaderClock.Load()
	if clock == nil || clock.id != sentFrom {
		return
	}
	now := int64(time.Since(leaderClockEpoch))
	// handlers run concurrently: an older message landing last must not move
	// the clock back, or the next gap comes out too long
	for {
		prev := clock.lastSeen.Load()
		if now <= prev {
			return
		}
		if clock.lastSeen.CompareAndSwap(prev, now) {
			metrics.RecordRaftLeaderSilence(time.Duration(now - prev))
			return
		}
	}
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
	r.recordLeaderChangeSilence(r.leaderClock.Swap(nil))
}

func (r *Raft) becomeLeader(ctx context.Context) {
	r.setLeaderState()

	// captured before the goroutine starts: ResignLeader nils r.Resign and a
	// later term replaces it, so the loop only uses its own
	resign := r.Resign
	heartbeatTicker := time.NewTicker(r.HeartbeatTimeout)
	leaderJobTicker := time.NewTicker(r.LeaderJobTimeout)

	go func() {
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
		err = r.LeaderJob(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to execute leader job")
		}

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
				err := r.sendHeartbeat(ctx, resign)
				if err != nil {
					log.Error().Err(err).Msg("failed to send heartbeat")
				}

			case <-leaderJobTicker.C:
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Msgf("recovered from panic in LeaderJob: %v", r)
						}
					}()
					// a resign can land after this tick was picked
					if resigned(resign) {
						return
					}
					err := r.LeaderJob(ctx)
					if err != nil {
						log.Error().Err(err).Msg("failed to execute leader job")
					}
				}()

			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				return
			}
		}
	}()
}

func (r *Raft) getRandomElectionTimeout() time.Duration {
	baseTimeout := r.HeartbeatTimeout * 5
	jitter := time.Duration(rand.Int63n(int64(r.HeartbeatTimeout * 5)))
	return baseTimeout + jitter
}

func (r *Raft) startElectionTimer() {
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

	if r.MissedHeartbeats < MaxMissedHeartbeats {
		r.MissedHeartbeats++
		log.Debug().Int("missed heartbeats", r.MissedHeartbeats).Msg("missed heartbeats")
		r.startElectionTimer()
		return
	}

	if !r.LastElectionTime.IsZero() && time.Since(r.LastElectionTime) < r.CooldownPeriod {
		log.Debug().Msg("Election cooldown period active, skipping election.")
		r.startElectionTimer()
		return
	}

	log.Debug().Msg("start election")
	metrics.RecordRaftElectionStarted(metrics.RaftElectionReasonTimeout)
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
