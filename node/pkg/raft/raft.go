package raft

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

const MESSAGE_BUFFER = 100
const HEARTBEAT_TIMEOUT = 100 * time.Millisecond

func NewRaftNode(node Node) *Raft {

	r := &Raft{
		Node:             node,
		Role:             "follower",
		VotedFor:         "",
		LeaderID:         "",
		VotesReceived:    0,
		Term:             0,
		Mutex:            sync.Mutex{},
		MessageBuffer:    make(chan Message, MESSAGE_BUFFER),
		Resign:           make(chan interface{}),
		HeartbeatTimeout: HEARTBEAT_TIMEOUT,
	}
	r.getTerm()
	// go r.run()
	return r
}

func (r *Raft) Run() {
	go r.subscribe(context.Background())
	r.startElectionTimer()
	var jobTicker <-chan time.Time
	if r.Node.GetJobTimeout() != nil {
		r.Node.SetJobTicker(r.Node.GetJobTimeout())
	}

	for {
		select {
		case msg := <-r.MessageBuffer:
			err := r.handleMessage(msg)
			if err != nil {
				log.Println("failed to handle message:", err)
			}
		case <-r.ElectionTimer.C:
			r.startElection()
		case <-jobTicker:
			if jobTicker != nil {
				r.Node.Job()
			}
		}
	}
}

func (r *Raft) getTerm() error {
	// load term from db
	loadedTerm := 0
	r.UpdateTerm(loadedTerm)
	return nil
}

func (r *Raft) saveTerm() error {
	// save term to db
	// term := r.GetCurrentTerm()
	return nil
}

func (r *Raft) subscribe(ctx context.Context) {
	sub, err := r.Node.GetTopic().Subscribe()
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

func (r *Raft) handleMessage(msg Message) error {
	switch msg.Type {
	case Heartbeat:
		return r.handleHeartbeat(msg)
	case RequestVote:
		return r.handleRequestVote(msg)
	case ReplyVote:
		return r.handleReplyVote(msg)
	default:
		return r.Node.HandleCustomMessage(msg)
	}
}

func (r *Raft) handleHeartbeat(msg Message) error {
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

	r.StopHeartbeatTicker()
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
		r.saveTerm()
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
		r.sendReplyVote(msg.SentFrom, false)
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

	r.sendReplyVote(msg.SentFrom, voteGranted)
	return nil
}

func (r *Raft) handleReplyVote(msg Message) error {
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
			r.becomeLeader()
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
	return r.Node.GetTopic().Publish(context.Background(), data)
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
	RequestVoteMessage := RequestVoteMessage{
		Term: r.GetCurrentTerm(),
	}
	marshalledRequestVoteMsg, err := json.Marshal(RequestVoteMessage)
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

func (r *Raft) StopHeartbeatTicker() {
	// should be called on leader job failure to resign and handover leadership
	if r.HeartbeatTicker != nil {
		r.HeartbeatTicker.Stop()
		r.HeartbeatTicker = nil
	}

	if r.Node.GetLeaderJobTicker() != nil {
		r.Node.GetLeaderJobTicker().Stop()
		r.Node.SetLeaderJobTicker(nil)
	}
	if r.Resign != nil {
		close(r.Resign)
		r.Resign = nil
	}
}

func (r *Raft) becomeLeader() {
	log.Println("Im now becoming leader")
	r.Resign = make(chan interface{})
	r.ElectionTimer.Stop()
	r.UpdateRole(Leader)
	r.HeartbeatTicker = time.NewTicker(r.HeartbeatTimeout)
	if r.Node.GetLeaderJobTimeout() != nil {
		r.Node.SetLeaderJobTicker(r.Node.GetLeaderJobTimeout())
	}

	go func() {
		for {
			select {
			case <-r.HeartbeatTicker.C:
				r.sendHeartbeat()
			case <-r.Node.GetLeaderJobTicker().C:
				// leader start its regular job
				r.Node.LeaderJob()
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
	r.saveTerm()
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
