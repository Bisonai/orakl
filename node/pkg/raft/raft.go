package raft

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const HEARTBEAT_TIMEOUT = 100 * time.Millisecond

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
					log.Error().Err(err).Str("Player", "Raft").Msg("failed to handle message")
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
	err := json.Unmarshal(msg.Data, &heartbeatMessage)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal heartbeat message")
		return err
	}

	if heartbeatMessage.LeaderID != msg.SentFrom {
		return errorSentinel.ErrRaftLeaderIdMismatch
	}

	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	currentRole := r.Role
	currentTerm := r.Term

	if heartbeatMessage.Term >= currentTerm {
		if currentRole == Leader {
			r.ResignLeader()
		}

		r.Term = max(heartbeatMessage.Term, currentTerm)
		r.Role = Follower
		r.LeaderID = heartbeatMessage.LeaderID
		r.startElectionTimer()
		return nil
	}

	return nil
}

func (r *Raft) handleRequestVote(ctx context.Context, msg Message) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.Role == Leader {
		return nil
	}

	var RequestVoteMessage RequestVoteMessage
	err := json.Unmarshal(msg.Data, &RequestVoteMessage)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal request vote message")
		return err
	}

	if RequestVoteMessage.Term > r.Term {
		r.Term = RequestVoteMessage.Term
		r.Role = Follower
		r.VotedFor = ""
	}

	if RequestVoteMessage.Term < r.Term {
		return r.sendReplyVote(ctx, msg.SentFrom, false)
	}

	if r.Role == Candidate && RequestVoteMessage.Term == r.Term && msg.SentFrom != r.GetHostId() {
		r.Role = Follower
		return r.sendReplyVote(ctx, msg.SentFrom, false)
	}

	voteGranted := false
	if r.VotedFor == "" || r.VotedFor == msg.SentFrom {
		voteGranted = true
		r.VotedFor = msg.SentFrom
	}
	log.Debug().Bool("vote granted", voteGranted).Msg("voted")
	return r.sendReplyVote(ctx, msg.SentFrom, voteGranted)
}

func (r *Raft) handleReplyVote(ctx context.Context, msg Message) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	if r.Role != Candidate {
		return nil
	}

	var replyVoteMessage ReplyRequestVoteMessage
	err := json.Unmarshal(msg.Data, &replyVoteMessage)
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
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.Topic.Publish(ctx, data)
}

func (r *Raft) sendHeartbeat(ctx context.Context) error {
	r.Mutex.Lock()
	heartbeatMessage := HeartbeatMessage{
		LeaderID: r.GetHostId(),
		Term:     r.Term,
	}
	r.Mutex.Unlock()
	marshalledHeartbeatMsg, err := json.Marshal(heartbeatMessage)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal heartbeat message")
		return err
	}

	message := Message{
		Type:     Heartbeat,
		SentFrom: r.GetHostId(),
		Data:     json.RawMessage(marshalledHeartbeatMsg),
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
	marshalledReplyVoteMsg, err := json.Marshal(replyVoteMessage)
	if err != nil {
		return err
	}
	message := Message{
		Type:     ReplyVote,
		SentFrom: r.GetHostId(),
		Data:     json.RawMessage(marshalledReplyVoteMsg),
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
	marshalledRequestVoteMsg, err := json.Marshal(requestVoteMessage)
	if err != nil {
		return err
	}

	message := Message{
		Type:     RequestVote,
		SentFrom: r.GetHostId(),
		Data:     json.RawMessage(marshalledRequestVoteMsg),
	}
	err = r.PublishMessage(ctx, message)
	if err != nil {
		return err
	}
	return nil
}

// utility functions

func (r *Raft) ResignLeader() {
	if r.Resign != nil {
		close(r.Resign)
		r.Resign = nil
		r.Role = Follower
		r.LeaderID = ""
		r.startElectionTimer()
	}
}

func (r *Raft) setLeaderState() {
	r.Resign = make(chan interface{})
	r.ElectionTimer.Stop()
	r.Role = Leader
	r.LeaderID = r.GetHostId()
	r.HeartbeatTicker = time.NewTicker(r.HeartbeatTimeout)
	r.LeaderJobTicker = time.NewTicker(r.LeaderJobTimeout)
}

func (r *Raft) becomeLeader(ctx context.Context) {
	r.setLeaderState()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Msgf("recovered from panic in leader job: %v", r)
			}
		}()

		for {
			select {
			case <-r.Resign:
				r.Mutex.Lock()
				r.HeartbeatTicker.Stop()
				r.LeaderJobTicker.Stop()
				r.Mutex.Unlock()
				return

			case <-r.HeartbeatTicker.C:
				err := r.sendHeartbeat(ctx)
				if err != nil {
					log.Error().Err(err).Msg("failed to send heartbeat")
				}

			case <-r.LeaderJobTicker.C:
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Msgf("recovered from panic in LeaderJob: %v", r)
						}
					}()
					err := r.LeaderJob(ctx)
					if err != nil {
						log.Error().Err(err).Msg("failed to execute leader job")
					}
				}()

			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				r.Mutex.Lock()
				r.HeartbeatTicker.Stop()
				r.LeaderJobTicker.Stop()
				r.Mutex.Unlock()
				return
			}
		}
	}()
}

func (r *Raft) getRandomElectionTimeout() time.Duration {
	baseTimeout := r.HeartbeatTimeout * 10
	jitter := time.Duration(rand.Int63n(int64(baseTimeout / 10))) // 10% jitter
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
	log.Debug().Msg("start election")
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Term++
	r.VotesReceived = 0
	r.Role = Candidate
	r.VotedFor = r.GetHostId()

	r.startElectionTimer()

	err := r.sendRequestVote(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to send request vote")
	}
}

func (r *Raft) unmarshalMessage(data []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(data, &m)
	if err != nil {
		return Message{}, err
	}
	return m, nil
}
