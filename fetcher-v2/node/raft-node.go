package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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
		return
	}

	n.stopHeartbeatTicker()
	n.startElectionTimer()

	if n.Data.Role != RoleTypes.Follower {
		n.Data.Role = RoleTypes.Follower
	}

	if n.Data.LeaderID != heartbeatMsg.LeaderID {
		n.Data.LeaderID = heartbeatMsg.LeaderID
	}
}

func (n *RaftNode) handleRequestVote(msg Message) {
	log.Println("receive vote request")
	if n.Data.Role == RoleTypes.Leader {
		// ignore vote request from other nodes
		return
	}

	if n.Data.Role == RoleTypes.Candidate {
		n.startElectionTimer()
	}

	voteGranted := false
	if n.Data.VotedFor == "" {
		voteGranted = true
		n.Data.VotedFor = msg.SentFrom
	}

	n.sendReplyVote(msg.SentFrom, voteGranted)
}

func (n *RaftNode) handleReplyVote(msg Message) {
	log.Println("receive vote reply")
	if n.Data.Role != RoleTypes.Candidate {
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
		n.Data.VotesReceived++
		if n.Data.VotesReceived > n.getSubscribersCount()/2 {
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
	log.Println("start election")
	// Transition to candidate state
	n.Data.Role = RoleTypes.Candidate
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
	// Construct RequestVote message
	message := Message{
		Type:     MessageTypes.RequestVote,
		SentFrom: n.Host.ID().String(),
		Data:     nil,
	}
	// Publish message
	err := n.publish(message)
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
	n.Data.Role = RoleTypes.Leader
	n.HeartbeatTicker = time.NewTicker(HEARTBEAT_TIMEOUT)
	submitTicker := time.NewTicker(SUBMIT_TIMEOUT)

	go func() {
		for {
			select {
			case <-n.HeartbeatTicker.C:
				n.sendHeartbeat()
			case <-submitTicker.C:
				log.Println("submit!")
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
