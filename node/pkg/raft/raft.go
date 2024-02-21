package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const HEARTBEAT_TIMEOUT = 100 * time.Millisecond

func NewRaftNode(h host.Host, ps *pubsub.PubSub, topic *pubsub.Topic, sub *pubsub.Subscription, messageBuffer int) *Raft {
	r := &Raft{
		Host:  h,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,

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
	var jobTicker <-chan time.Time
	if node.GetJobTimeout() != nil {
		err := node.SetJobTicker(node.GetJobTimeout())
		if err != nil {
			log.Println("failed to set job ticker")
		}
		jobTicker = node.GetJobTicker().C
	}

	for {
		select {
		case msg := <-r.MessageBuffer:
			err := r.handleMessage(node, msg)
			if err != nil {
				log.Println("failed to handle message:", err)
			}
		case <-r.ElectionTimer.C:
			r.startElection()
		case <-jobTicker:
			err := node.Job()
			if err != nil {
				log.Println("failed to execute job:", err)
			}
		}
	}
}

func (r *Raft) subscribe(ctx context.Context) {
	sub, err := r.Topic.Subscribe()
	if err != nil {
		log.Println("failed to subscribe to topic")
	}
	for {
		rawMsg, err := sub.Next(ctx)
		if err != nil {
			log.Println("failed to get message from topic")
		}
		msg, err := r.unmarshalMessage(rawMsg.Data)
		if err != nil {
			log.Println("failed to unmarshal message")
		}
		r.MessageBuffer <- msg
	}
}

// handler for incoming messages

func (r *Raft) handleMessage(node Node, msg Message) error {
	switch msg.Type {
	case Heartbeat:
		return r.handleHeartbeat(node, msg)
	case RequestVote:
		return r.handleRequestVote(msg)
	case ReplyVote:
		return r.handleReplyVote(node, msg)
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
		return err
	}

	if heartbeatMessage.LeaderID != msg.SentFrom {
		return fmt.Errorf("leader id mismatch")
	}

	r.StopHeartbeatTicker(node)
	r.startElectionTimer()

	currentRole := r.GetRole()
	currentTerm := r.GetCurrentTerm()
	currentLeader := r.GetLeader()

	// If the current role is Candidate or the current role is Leader and the current term is less than the heartbeat term, update the role to Follower
	shouldUpdateRoleToFollower := (currentRole == Candidate) || (currentRole == Leader && currentTerm < heartbeatMessage.Term)
	if shouldUpdateRoleToFollower {
		r.UpdateRole(Follower)
	}

	// If the heartbeat term is greater than the current term, update the term
	if heartbeatMessage.Term > currentTerm {
		r.UpdateTerm(heartbeatMessage.Term)
	}

	// If the current leader is not the leader in the heartbeat message and the current term is less than the heartbeat term, update the leader
	shouldUpdateLeader := (currentLeader != heartbeatMessage.LeaderID) && (currentTerm < heartbeatMessage.Term)
	if shouldUpdateLeader {
		r.UpdateLeader(heartbeatMessage.LeaderID)
	}
	return nil
}

func (r *Raft) handleRequestVote(msg Message) error {
	log.Println("received request vote message")
	if r.GetRole() == Leader {
		return nil
	}

	var RequestVoteMessage RequestVoteMessage
	err := json.Unmarshal(msg.Data, &RequestVoteMessage)
	if err != nil {
		log.Println("failed to unmarshal request vote message:", err)
		return err
	}

	if RequestVoteMessage.Term > r.GetCurrentTerm() {
		r.UpdateTerm(RequestVoteMessage.Term)
	}

	if RequestVoteMessage.Term < r.GetCurrentTerm() {
		err := r.sendReplyVote(msg.SentFrom, false)
		if err != nil {
			log.Println("failed to send reply vote:", err)
		}
		return nil
	}

	if r.GetRole() == Candidate {
		r.startElectionTimer()
	}

	voteGranted := false
	if r.GetVotedFor() == "" || r.GetVotedFor() == msg.SentFrom {
		voteGranted = true
		r.UpdateVotedFor(msg.SentFrom)
	}

	return r.sendReplyVote(msg.SentFrom, voteGranted)
}

func (r *Raft) handleReplyVote(node Node, msg Message) error {
	log.Println("received reply vote message")
	if r.GetRole() != Candidate {
		return nil
	}

	var replyVoteMessage ReplyRequestVoteMessage
	err := json.Unmarshal(msg.Data, &replyVoteMessage)
	if err != nil {
		return err
	}

	if replyVoteMessage.VoteGranted && replyVoteMessage.LeaderID == r.GetHostId() {
		r.IncreaseVote()
		log.Println("vote received:", r.GetVoteReceived())
		log.Println("subscribers count:", r.SubscribersCount())
		if r.GetVoteReceived() >= (r.SubscribersCount()+1)/2 {
			r.becomeLeader(node)
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
	_heartbeatMessage := HeartbeatMessage{
		LeaderID: r.GetHostId(),
		Term:     r.GetCurrentTerm(),
	}
	marshalledHeartbeatMsg, err := json.Marshal(_heartbeatMessage)
	if err != nil {
		log.Println("failed to marshal heartbeat message")
		return err
	}

	heartbeatMessage := Message{
		Type:     Heartbeat,
		SentFrom: r.GetHostId(),
		Data:     json.RawMessage(marshalledHeartbeatMsg),
	}
	err = r.PublishMessage(heartbeatMessage)
	if err != nil {
		log.Println("failed to send heartbeat")
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

	if node.GetLeaderJobTicker() != nil {
		node.GetLeaderJobTicker().Stop()
		err := node.SetLeaderJobTicker(nil)
		if err != nil {
			log.Println("failed to stop leader job ticker")
		}
	}
	if r.Resign != nil {
		close(r.Resign)
		r.Resign = nil
	}
}

func (r *Raft) becomeLeader(node Node) {
	log.Println("Im now becoming leader")
	r.Resign = make(chan interface{})
	r.ElectionTimer.Stop()
	r.UpdateRole(Leader)
	r.HeartbeatTicker = time.NewTicker(r.HeartbeatTimeout)

	var leaderJobTicker <-chan time.Time
	if node.GetLeaderJobTimeout() != nil {
		err := node.SetLeaderJobTicker(node.GetLeaderJobTimeout())
		if err != nil {
			log.Println("failed to set leader job ticker")
		}
		leaderJobTicker = node.GetLeaderJobTicker().C
	}

	go func() {
		for {
			select {
			case <-r.HeartbeatTicker.C:
				err := r.sendHeartbeat()
				if err != nil {
					log.Println("failed to send heartbeat:", err)
				}
			case <-leaderJobTicker:
				err := node.LeaderJob()
				if err != nil {
					log.Println("failed to execute leader job:", err)
				}
			case <-r.Resign:
				log.Println("resigning as leader")
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
	log.Println("start election")

	r.UpdateRole(Candidate)

	r.startElectionTimer()

	err := r.sendRequestVote()
	if err != nil {
		log.Println("failed to send request vote")
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
