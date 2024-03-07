package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const HEARTBEAT_TIMEOUT = 100 * time.Millisecond

func NewRaftNode(h host.Host, ps *pubsub.PubSub, topic *pubsub.Topic, messageBuffer int) *Raft {
	r := &Raft{
		Host:  h,
		Ps:    ps,
		Topic: topic,

		Role:             "follower",
		VotedFor:         "",
		LeaderID:         "",
		VotesReceived:    0,
		Term:             0,
		Mutex:            sync.Mutex{},
		MessageBuffer:    make(chan Message, messageBuffer),
		Resign:           make(chan interface{}),
		HeartbeatTimeout: HEARTBEAT_TIMEOUT,
	}
	return r
}

func (r *Raft) Run(ctx context.Context, node Node) {
	go r.subscribe(ctx)
	r.startElectionTimer()

	for {
		select {
		case msg := <-r.MessageBuffer:
			err := r.handleMessage(ctx, node, msg)
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

func (r *Raft) handleMessage(ctx context.Context, node Node, msg Message) error {
	switch msg.Type {
	case Heartbeat:
		return r.handleHeartbeat(node, msg)
	case RequestVote:
		return r.handleRequestVote(msg)
	case ReplyVote:
		return r.handleReplyVote(ctx, node, msg)
	default:
		return node.HandleCustomMessage(msg)
	}
}

func (r *Raft) handleHeartbeat(node Node, msg Message) error {
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
		return fmt.Errorf("leader id mismatch")
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
		r.StopHeartbeatTicker(node)
		r.UpdateRole(Follower)
	} else if currentRole == Candidate {
		r.UpdateRole(Follower)
	}

	r.startElectionTimer()
	r.UpdateTerm(heartbeatMessage.Term)

	if currentLeader != heartbeatMessage.LeaderID {
		r.UpdateLeader(heartbeatMessage.LeaderID)
	}

	return nil
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
		// Deny the vote and revert to follower
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

func (r *Raft) handleReplyVote(ctx context.Context, node Node, msg Message) error {
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
			r.becomeLeader(ctx, node)
		}
	}
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

func (r *Raft) StopHeartbeatTicker(node Node) {
	// should be called on leader job failure to resign and handover leadership
	if r.HeartbeatTicker != nil {
		r.HeartbeatTicker.Stop()
		r.HeartbeatTicker = nil
	}

	if r.Resign != nil {
		close(r.Resign)
		r.Resign = nil
	}
}

func (r *Raft) becomeLeader(ctx context.Context, node Node) {
	log.Debug().Msg("becoming leader")

	r.Resign = make(chan interface{})
	r.ElectionTimer.Stop()
	r.UpdateRole(Leader)
	r.HeartbeatTicker = time.NewTicker(r.HeartbeatTimeout)

	leaderJobTimer := time.NewTimer(*node.GetLeaderJobTimeout())

	go func() {
		for {
			select {
			case <-r.HeartbeatTicker.C:
				err := r.sendHeartbeat()
				if err != nil {
					log.Error().Err(err).Msg("failed to send heartbeat")
				}
			case <-leaderJobTimer.C:
				err := node.LeaderJob()
				if err != nil {
					log.Error().Err(err).Msg("failed to execute leader job")
				}
				leaderJobTimer.Reset(*node.GetLeaderJobTimeout())

			case <-r.Resign:
				log.Debug().Msg("resigning as leader")
				leaderJobTimer.Stop()
				return
			case <-ctx.Done():
				log.Debug().Msg("context cancelled")
				leaderJobTimer.Stop()
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
