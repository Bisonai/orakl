package node

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
	"github.com/libp2p/go-libp2p/core/peer"
)

type MessageType struct {
	Heartbeat      string
	RequestVote    string
	ReplyHeartbeat string
	ReplyVote      string
}

var MessageTypes = MessageType{
	Heartbeat:      "heartbeat",
	RequestVote:    "requestVote",
	ReplyHeartbeat: "replyHeartbeat",
	ReplyVote:      "replyVote",
}

type RoleType struct {
	Leader    string
	Candidate string
	Follower  string
}

var RoleTypes = RoleType{
	Leader:    "leader",
	Candidate: "candidate",
	Follower:  "follower",
}

const HEARTBEAT_TIMEOUT = 150 * time.Millisecond
const SUBMIT_TIMEOUT = 10 * time.Second

func ElectionTimeout() time.Duration {
	minTimeout := int(HEARTBEAT_TIMEOUT) * 10
	maxTimeout := int(HEARTBEAT_TIMEOUT) * 15
	return time.Duration(minTimeout + rand.Intn(maxTimeout-minTimeout))
}

// only taking leader election from raft for now

type RaftState struct {
	Role          string
	VotedFor      string
	LeaderID      string
	VotesReceived int
	Term          int
	Mutex         sync.Mutex
}

type PubSubComponents struct {
	Ps    *pubsub.PubSub
	Topic *pubsub.Topic
	Sub   *pubsub.Subscription
}

type RaftNode struct {
	Host            host.Host
	PubSub          PubSubComponents
	Data            RaftState
	Msg             chan Message
	HeartbeatTicker *time.Ticker
	ElectionTimer   *time.Timer
	Resign          chan interface{}
}

type Message struct {
	Type     string          `json:"type"`
	SentFrom string          `json:"sentFrom"`
	Data     json.RawMessage `json:"data"`
}

type HeartbeatMessage struct {
	LeaderID string `json:"leaderID"`
	Term     int    `json:"term"`
}

type RequestVoteMessage struct {
	Term int `json:"term"`
}

type ReplyRequestVoteMessage struct {
	VoteGranted bool   `json:"voteGranted"`
	LeaderID    string `json:"leaderID"`
}

func NewRaftNode(host host.Host, ps *pubsub.PubSub, topicString string) (*RaftNode, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &RaftNode{
		Host:   host,
		PubSub: PubSubComponents{Ps: ps, Topic: topic, Sub: sub},
		Data: RaftState{
			VotedFor:      "",
			Role:          RoleTypes.Follower,
			LeaderID:      "",
			VotesReceived: 0,
			Term:          0,
		},
		Msg:    make(chan Message, 15),
		Resign: make(chan interface{}),
	}, nil
}

func (n *RaftNode) unmarshalMessage(data []byte) (Message, error) {
	var message Message
	err := json.Unmarshal(data, &message)
	if err != nil {
		return Message{}, err
	}
	return message, nil
}

func (n *RaftNode) unmarshalMessageData(data json.RawMessage, messageType string) (interface{}, error) {
	switch messageType {
	case MessageTypes.Heartbeat:
		var entry HeartbeatMessage
		err := json.Unmarshal(data, &entry)
		if err != nil {
			return HeartbeatMessage{}, err
		}
		return entry, nil
	case MessageTypes.RequestVote:
		var vote RequestVoteMessage
		err := json.Unmarshal(data, &vote)
		if err != nil {
			return RequestVoteMessage{}, err
		}
		return vote, nil
	case MessageTypes.ReplyVote:
		var replyVote ReplyRequestVoteMessage
		err := json.Unmarshal(data, &replyVote)
		if err != nil {
			return ReplyRequestVoteMessage{}, err
		}
		return replyVote, nil
	default:
		return nil, fmt.Errorf("unexpected message type")
	}
}

func (n *RaftNode) subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down subscription")
			return
		default:
			rawMsg, err := n.PubSub.Sub.Next(ctx)
			if err != nil {
				log.Println("message receive failed:" + err.Error())
				continue
			}
			message, err := n.unmarshalMessage(rawMsg.Data)
			if err != nil {
				log.Println("unexpected message:" + err.Error())
				continue
			}

			n.Msg <- message
		}
	}
}

func (n *RaftNode) Run() {
	go n.subscribe(context.Background())
	n.startElectionTimer()

	for {
		select {
		case msg := <-n.Msg:
			switch msg.Type {
			case MessageTypes.Heartbeat:
				n.handleHeartbeat(msg)
			case MessageTypes.RequestVote:
				n.handleRequestVote(msg)
			case MessageTypes.ReplyVote:
				n.handleReplyVote(msg)
			default:
				log.Println("unexpected message type")
			}
		case <-n.ElectionTimer.C:
			n.startElection()
		}

	}
}

func (n *RaftNode) handleHeartbeat(msg Message) {
	if msg.SentFrom == n.Host.ID().String() {
		return
	}

	heartbeat, err := n.unmarshalMessageData(msg.Data, msg.Type)
	if err != nil {
		log.Println("failed to unmarshal heartbeat message:" + err.Error())
		return
	}
	heartbeatMsg := heartbeat.(HeartbeatMessage)

	if heartbeatMsg.LeaderID != msg.SentFrom {
		// invalid message
		return
	}

	n.stopHeartbeatTicker()
	n.startElectionTimer()
	currentRole := n.getCurrentRole()
	currentTerm := n.getCurrentTerm()
	currentLeader := n.getCurrentLeader()

	if currentRole == RoleTypes.Candidate {
		n.updateRole(RoleTypes.Follower)
	}
	if currentRole == RoleTypes.Leader && currentTerm < heartbeatMsg.Term {
		n.updateRole(RoleTypes.Follower)
	}

	if heartbeatMsg.Term > currentTerm {
		n.updateTerm(heartbeatMsg.Term)
	}

	if currentLeader != heartbeatMsg.LeaderID && currentTerm < heartbeatMsg.Term {
		n.updateLeader(heartbeatMsg.LeaderID)
	}
}

func (n *RaftNode) handleRequestVote(msg Message) {
	log.Println("receive vote request")

	currentRole := n.getCurrentRole()
	votedFor := n.getCurrentVotedFor()

	if currentRole == RoleTypes.Leader {
		// ignore vote request from other nodes
		return
	}

	requestVote, err := n.unmarshalMessageData(msg.Data, msg.Type)
	if err != nil {
		log.Println("failed to unmarshal vote request message:" + err.Error())
	}
	requestVoteMsg := requestVote.(RequestVoteMessage)
	if requestVoteMsg.Term < n.getCurrentTerm() {
		n.updateTerm(requestVoteMsg.Term)
	}
	// should reject vote request if term is lower, but for now just ignore it

	if currentRole == RoleTypes.Candidate {
		n.startElectionTimer()
	}

	voteGranted := false
	if votedFor == "" {
		voteGranted = true
		n.updateVotedFor(msg.SentFrom)
	}

	n.sendReplyVote(msg.SentFrom, voteGranted)
}

func (n *RaftNode) handleReplyVote(msg Message) {
	log.Println("receive vote reply")
	if n.getCurrentRole() != RoleTypes.Candidate {
		// ignore vote reply from other nodes
		return
	}

	replyVote, err := n.unmarshalMessageData(msg.Data, msg.Type)
	if err != nil {
		log.Println("failed to unmarshal vote reply message:" + err.Error())
		return
	}
	replyVoteMsg := replyVote.(ReplyRequestVoteMessage)

	if replyVoteMsg.VoteGranted && replyVoteMsg.LeaderID == n.Host.ID().String() {
		log.Println("vote granted")
		newVotes := n.getCurrentVotes() + 1
		n.updateVoteReceived(newVotes)
		if newVotes > n.getSubscribersCount()/2 {
			n.becomeLeader()
		}
	}
}

func (n *RaftNode) startElectionTimer() {
	if n.ElectionTimer != nil {
		n.ElectionTimer.Stop()
	}

	n.ElectionTimer = time.NewTimer(ElectionTimeout())
}

func (n *RaftNode) startElection() {
	n.updateTerm(n.getCurrentTerm() + 1)
	n.updateVoteReceived(0)
	log.Println("start election")
	// Transition to candidate state
	n.updateRole(RoleTypes.Candidate)
	// Reset election timer
	n.startElectionTimer()
	// Send RequestVote RPCs to all other servers
	log.Println("sent request vote")
	err := n.sendRequestVote()
	if err != nil {
		log.Println("failed to send request vote message:" + err.Error())
	}
}

func (n *RaftNode) publish(message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return n.PubSub.Topic.Publish(context.Background(), data)
}

func (n *RaftNode) sendRequestVote() error {
	RequestVoteMessage := RequestVoteMessage{
		Term: n.getCurrentTerm(),
	}
	marshalledRequestVoteMsg, err := json.Marshal(RequestVoteMessage)
	if err != nil {
		return err
	}

	// Construct RequestVote message
	message := Message{
		Type:     MessageTypes.RequestVote,
		SentFrom: n.Host.ID().String(),
		Data:     json.RawMessage(marshalledRequestVoteMsg),
	}
	// Publish message
	err = n.publish(message)
	if err != nil {
		return err
	}
	return nil
}

func (n *RaftNode) sendReplyVote(to string, voteGranted bool) error {
	replyVoteMessage := ReplyRequestVoteMessage{
		VoteGranted: voteGranted,
		LeaderID:    to,
	}
	marshalledReplyVoteMsg, err := json.Marshal(replyVoteMessage)
	if err != nil {
		return err
	}
	// Construct ReplyVote message
	message := Message{
		Type:     MessageTypes.ReplyVote,
		SentFrom: n.Host.ID().String(),
		Data:     json.RawMessage(marshalledReplyVoteMsg),
	}
	// Publish message
	err = n.publish(message)
	if err != nil {
		return err
	}
	return nil
}

func (n *RaftNode) sendHeartbeat() error {
	heartbeatMessage := HeartbeatMessage{
		LeaderID: n.Host.ID().String(),
	}
	marshalledHeartbeatMsg, err := json.Marshal(heartbeatMessage)
	if err != nil {
		return err
	}
	// Construct heartbeat message
	message := Message{
		Type:     MessageTypes.Heartbeat,
		SentFrom: n.Host.ID().String(),
		Data:     json.RawMessage(marshalledHeartbeatMsg),
	}
	// Publish message
	err = n.publish(message)
	if err != nil {
		return err
	}
	return nil
}

func (n *RaftNode) becomeLeader() {
	log.Printf("(%s) I am leader", n.Host.ID().String())
	n.Resign = make(chan interface{})
	n.ElectionTimer.Stop()
	n.updateRole(RoleTypes.Leader)
	n.HeartbeatTicker = time.NewTicker(HEARTBEAT_TIMEOUT)
	submitTicker := time.NewTicker(SUBMIT_TIMEOUT)

	go func() {
		for {
			select {
			case <-n.HeartbeatTicker.C:
				n.sendHeartbeat()
			case <-submitTicker.C:
				n.submit()
			case <-n.Resign:
				log.Println("I resign leader")
				return
			}
		}
	}()
}

func (n *RaftNode) stopHeartbeatTicker() {
	if n.HeartbeatTicker != nil {
		n.HeartbeatTicker.Stop()
		n.HeartbeatTicker = nil
	}
	if n.Resign != nil {
		close(n.Resign)
		n.Resign = nil
	}
}

func (n *RaftNode) getSubscribersCount() int {
	peers := n.subscribers()
	return len(peers)
}

func (n *RaftNode) subscribers() []peer.ID {
	return n.PubSub.Ps.ListPeers(n.PubSub.Topic.String())
}

func (n *RaftNode) updateTerm(newTerm int) error {
	if newTerm < n.Data.Term {
		return fmt.Errorf("invalid term")
	}
	n.Data.Mutex.Lock()
	n.Data.Term = newTerm
	n.Data.VotedFor = ""
	n.Data.Mutex.Unlock()
	return nil
}

func (n *RaftNode) updateLeader(leader string) {
	n.Data.Mutex.Lock()
	n.Data.LeaderID = leader
	n.Data.Mutex.Unlock()
}

func (n *RaftNode) updateVoteReceived(votes int) {
	n.Data.Mutex.Lock()
	n.Data.VotesReceived = votes
	n.Data.Mutex.Unlock()
}

func (n *RaftNode) updateRole(role string) {
	n.Data.Mutex.Lock()
	n.Data.Role = role
	n.Data.Mutex.Unlock()
}

func (n *RaftNode) updateVotedFor(votedFor string) {
	n.Data.Mutex.Lock()
	n.Data.VotedFor = votedFor
	n.Data.Mutex.Unlock()
}

func (n *RaftNode) getCurrentRole() string {
	n.Data.Mutex.Lock()
	role := n.Data.Role
	n.Data.Mutex.Unlock()
	return role
}

func (n *RaftNode) getCurrentTerm() int {
	n.Data.Mutex.Lock()
	term := n.Data.Term
	n.Data.Mutex.Unlock()
	return term
}

func (n *RaftNode) getCurrentVotes() int {
	n.Data.Mutex.Lock()
	votes := n.Data.VotesReceived
	n.Data.Mutex.Unlock()
	return votes
}

func (n *RaftNode) getCurrentVotedFor() string {
	n.Data.Mutex.Lock()
	votedFor := n.Data.VotedFor
	n.Data.Mutex.Unlock()
	return votedFor
}

func (n *RaftNode) getCurrentLeader() string {
	n.Data.Mutex.Lock()
	leader := n.Data.LeaderID
	n.Data.Mutex.Unlock()
	return leader
}

func (n *RaftNode) submit() {
	n.updateTerm(n.Data.Term + 1)
	log.Println("submit!")
}
