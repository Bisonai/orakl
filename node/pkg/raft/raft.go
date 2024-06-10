package raft

import (
	"context"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/utils/set"
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

		MessageBuffer:    make(chan Message, messageBuffer),
		Resign:           make(chan interface{}),
		HeartbeatTimeout: HEARTBEAT_TIMEOUT,

		LeaderJobTimeout: leaderJobTimeout,

		PrevPeers: *set.NewSet[string](),
		Peers:     *set.NewSet[string](),
	}
	return r
}

func (r *Raft) Run(ctx context.Context) {
	go r.subscribe(ctx)
	r.startElectionTimer()

	for {
		select {
		case msg := <-r.MessageBuffer:
			err := r.handleMessage(ctx, msg)
			if err != nil {
				log.Error().Err(err).Msg("failed to handle message")
			}

		case <-r.ElectionTimer.C:
			r.startElection()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Raft) subscribe(ctx context.Context) {
	sub, err := r.Topic.Subscribe()
	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to topic")
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
			msg, err := r.unmarshalMessage(rawMsg.Data)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal message")
				continue
			}
			r.MessageBuffer <- msg
		}
	}
}

// handler for incoming messages

func (r *Raft) handleMessage(ctx context.Context, msg Message) error {
	switch msg.Type {
	case Heartbeat:
		return r.handleHeartbeat(msg)
	case RequestVote:
		return r.handleRequestVote(msg)
	case ReplyVote:
		return r.handleReplyVote(ctx, msg)
	case ReplyHeartbeat:
		return r.handleReplyHeartbeat(msg)
	default:
		return r.HandleCustomMessage(ctx, msg)
	}
}

func (r *Raft) handleHeartbeat(msg Message) error {
	r.Peers = r.PrevPeers
	r.PrevPeers = *set.NewSet[string]()

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

	currentRole := r.GetRole()
	currentTerm := r.GetCurrentTerm()
	currentLeader := r.GetLeader()

	if currentTerm > heartbeatMessage.Term && currentRole != Leader {
		r.startElectionTimer()
		return nil
	}

	if currentTerm > heartbeatMessage.Term && currentRole == Leader {
		return nil
	}

	if currentRole == Leader {
		r.ResignLeader()
	} else if currentRole == Candidate {
		r.UpdateRole(Follower)
	}

	r.startElectionTimer()
	r.UpdateTerm(heartbeatMessage.Term)

	if currentLeader != heartbeatMessage.LeaderID {
		r.UpdateLeader(heartbeatMessage.LeaderID)
	}

	return r.sendReplyHeartbeat()
}

func (r *Raft) handleRequestVote(msg Message) error {
	if r.GetRole() == Leader {
		return nil
	}

	var RequestVoteMessage RequestVoteMessage
	err := json.Unmarshal(msg.Data, &RequestVoteMessage)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal request vote message")
		return err
	}

	currentTerm := r.GetCurrentTerm()

	if RequestVoteMessage.Term > currentTerm {
		r.UpdateTerm(RequestVoteMessage.Term)
	}

	if RequestVoteMessage.Term < currentTerm {
		return r.sendReplyVote(msg.SentFrom, false)
	}

	if r.GetRole() == Candidate && RequestVoteMessage.Term == currentTerm && msg.SentFrom != r.GetHostId() {
		r.UpdateRole(Follower)
		return r.sendReplyVote(msg.SentFrom, false)
	}

	voteGranted := false
	if r.GetVotedFor() == "" || r.GetVotedFor() == msg.SentFrom {
		voteGranted = true
		r.UpdateVotedFor(msg.SentFrom)
	}
	log.Debug().Bool("vote granted", voteGranted).Msg("voted")
	return r.sendReplyVote(msg.SentFrom, voteGranted)
}

func (r *Raft) handleReplyVote(ctx context.Context, msg Message) error {
	if r.GetRole() != Candidate {
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

	if replyVoteMessage.VoteGranted && replyVoteMessage.LeaderID == r.GetHostId() && r.GetRole() == Candidate {
		r.IncreaseVote()
		log.Debug().Int("vote received", r.GetVoteReceived()).Msg("vote received")
		log.Debug().Int("subscribers count", r.SubscribersCount()).Msg("subscribers count")
		if r.GetVoteReceived() >= (r.SubscribersCount()+1)/2 {
			r.becomeLeader(ctx)
		}
	}
	return nil
}

func (r *Raft) handleReplyHeartbeat(msg Message) error {
	var replyHeartbeatMessage ReplyHeartbeatMessage
	err := json.Unmarshal(msg.Data, &replyHeartbeatMessage)
	if err != nil {
		return err
	}

	r.PrevPeers.Add(msg.SentFrom)
	return nil
}

// publishing messages

func (r *Raft) PublishMessage(msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.Topic.Publish(context.Background(), data)
}

func (r *Raft) sendHeartbeat() error {

	heartbeatMessage := HeartbeatMessage{
		LeaderID: r.GetHostId(),
		Term:     r.GetCurrentTerm(),
	}
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
	err = r.PublishMessage(message)
	if err != nil {
		log.Error().Err(err).Msg("failed to send heartbeat")
		return err
	}
	return nil
}

func (r *Raft) sendReplyHeartbeat() error {
	replyHeartbeatMessage := ReplyHeartbeatMessage{}
	marshalledReplyHeartbeatMsg, err := json.Marshal(replyHeartbeatMessage)
	if err != nil {
		return err
	}
	message := Message{
		Type:     ReplyHeartbeat,
		SentFrom: r.GetHostId(),
		Data:     json.RawMessage(marshalledReplyHeartbeatMsg),
	}
	err = r.PublishMessage(message)
	if err != nil {
		return err
	}
	return nil
}

func (r *Raft) sendReplyVote(to string, voteGranted bool) error {
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
	err = r.PublishMessage(message)
	if err != nil {
		return err
	}
	return nil
}

func (r *Raft) sendRequestVote() error {
	requestVoteMessage := RequestVoteMessage{
		Term: r.GetCurrentTerm(),
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
	err = r.PublishMessage(message)
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

		r.UpdateRole(Follower)
		r.UpdateLeader("")
		r.startElectionTimer()
	}
}

func (r *Raft) becomeLeader(ctx context.Context) {

	log.Debug().Msg("becoming leader")

	r.Resign = make(chan interface{})
	r.ElectionTimer.Stop()
	r.UpdateRole(Leader)
	r.UpdateLeader(r.GetHostId())
	r.HeartbeatTicker = time.NewTicker(r.HeartbeatTimeout)
	r.LeaderJobTicker = time.NewTicker(r.LeaderJobTimeout)

	go func() {
		for {
			select {
			case <-r.Resign:
				log.Debug().Msg("resigning as leader")
				r.HeartbeatTicker.Stop()
				r.LeaderJobTicker.Stop()

				return

			case <-r.HeartbeatTicker.C:
				err := r.sendHeartbeat()
				if err != nil {
					log.Error().Err(err).Msg("failed to send heartbeat")
				}

			case <-r.LeaderJobTicker.C:
				go func() {
					err := r.LeaderJob()
					if err != nil {
						log.Error().Err(err).Msg("failed to execute leader job")
					}
				}()

			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				r.HeartbeatTicker.Stop()
				r.LeaderJobTicker.Stop()
				return
			}
		}
	}()
}

func (r *Raft) getRandomElectionTimeout() time.Duration {
	minTimeout := int(r.HeartbeatTimeout) * 3
	maxTimeout := int(r.HeartbeatTimeout) * 6
	return time.Duration(minTimeout + rand.Intn(maxTimeout-minTimeout))
}

func (r *Raft) startElectionTimer() {
	if r.ElectionTimer != nil {
		r.ElectionTimer.Stop()
	}
	r.ElectionTimer = time.NewTimer(r.getRandomElectionTimeout())
}

func (r *Raft) startElection() {
	r.IncreaseTerm()
	r.UpdateVoteReceived(0)
	log.Debug().Msg("start election")

	r.UpdateRole(Candidate)
	r.UpdateVotedFor(r.GetHostId())

	r.startElectionTimer()

	err := r.sendRequestVote()
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
