//nolint:all
package raft

import (
	"context"
	"fmt"
	"testing"
	"time"

	libp2psetup "bisonai.com/orakl/node/pkg/libp2p/setup"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/assert"
)

type TestItems struct {
	Cancel    context.CancelFunc
	Hosts     []host.Host
	Pss       []*pubsub.PubSub
	Topics    []*pubsub.Topic
	RaftNodes []*Raft
}

const topicString = "orakl-raft-test-topic"

func setup(ctx context.Context, cancel context.CancelFunc) (func() error, *TestItems, error) {

	hosts := make([]host.Host, 0)
	for i := 0; i < 3; i++ {
		host, err := libp2psetup.NewHost(ctx)
		if err != nil {
			return nil, nil, err
		}
		hosts = append(hosts, host)
	}

	pss := make([]*pubsub.PubSub, 0)
	for i := 0; i < 3; i++ {
		ps, err := libp2psetup.MakePubsub(ctx, hosts[i])
		if err != nil {
			return nil, nil, err
		}
		pss = append(pss, ps)
	}

	for i := 0; i < 2; i++ {
		host := hosts[i]
		host.Connect(ctx, peer.AddrInfo{ID: hosts[i+1].ID(), Addrs: hosts[i+1].Addrs()})
	}

	topics := make([]*pubsub.Topic, 0)
	for i := 0; i < 3; i++ {
		topic, err := pss[i].Join(topicString)
		if err != nil {
			return nil, nil, err
		}
		topics = append(topics, topic)
	}

	raftNodes := []*Raft{}
	for i := 0; i < 3; i++ {
		node := NewRaftNode(hosts[i], pss[i], topics[i], 100, 500*time.Millisecond)
		node.LeaderJob = func() error {
			log.Debug().Int("subscribers", node.SubscribersCount()).Int("Term", node.GetCurrentTerm()).Msg("Leader job")
			node.IncreaseTerm()
			return nil
		}
		node.HandleCustomMessage = func(ctx context.Context, message Message) error {
			log.Debug().Msg("Custom message")
			return fmt.Errorf("Unknow message type")
		}
		raftNodes = append(raftNodes, node)
	}

	testItems := &TestItems{
		Cancel:    cancel,
		Hosts:     hosts,
		Pss:       pss,
		Topics:    topics,
		RaftNodes: raftNodes,
	}

	cleanup := func() error {
		testItems.Cancel()
		for _, topic := range testItems.Topics {
			topic.Close()
		}
		for _, host := range testItems.Hosts {
			host.Close()
		}
		return nil
	}

	return cleanup, testItems, nil
}

func TestRaft(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	cleanup, testItems, err := setup(ctx, cancel)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	// give time to connect and start
	time.Sleep(1 * time.Second)

	// all raft nodes should have same leader
	leaderIds := make(map[string]any)
	for i := range testItems.RaftNodes {
		leader := testItems.RaftNodes[i].GetLeader()
		leaderIds[leader] = struct{}{}
	}
	assert.Equal(t, 1, len(leaderIds))

	// every node should be follower except leader
	follwerCount := 0
	for i := range testItems.RaftNodes {
		if testItems.RaftNodes[i].GetRole() == Follower {
			follwerCount++
		}
	}
	assert.Equal(t, len(testItems.RaftNodes)-1, follwerCount)

	// all raft nodes should have same term
	terms := make(map[int]any)
	for i := range testItems.RaftNodes {
		term := testItems.RaftNodes[i].GetCurrentTerm()
		terms[term] = struct{}{}
	}
	assert.Equal(t, 1, len(terms))

	// Term should be increasing every second based on leader job
	termsBefore := []int{}
	for i := range testItems.RaftNodes {
		termsBefore = append(termsBefore, testItems.RaftNodes[i].GetCurrentTerm())
	}
	time.Sleep(1 * time.Second)
	termsAfter := []int{}
	for i := range testItems.RaftNodes {
		termsAfter = append(termsAfter, testItems.RaftNodes[i].GetCurrentTerm())
	}

	for i := range termsBefore {
		assert.Greater(t, termsAfter[i], termsBefore[i])
	}

	// If leader resigns, a leader should be elected
	for i := range testItems.RaftNodes {
		if testItems.RaftNodes[i].GetRole() == Leader {
			testItems.RaftNodes[i].ResignLeader()
			break
		}
	}
	time.Sleep(1 * time.Second)
	leaderIds = make(map[string]any)
	for i := range testItems.RaftNodes {
		leader := testItems.RaftNodes[i].GetLeader()
		leaderIds[leader] = struct{}{}
	}
	assert.Equal(t, 1, len(leaderIds))

	// If new node joins, it should also be working as expected
	newHost, err := libp2psetup.NewHost(ctx)
	if err != nil {
		t.Fatalf("error making host: %v", err)
	}
	newHost.Connect(ctx, peer.AddrInfo{ID: testItems.Hosts[0].ID(), Addrs: testItems.Hosts[0].Addrs()})
	defer newHost.Close()
	newPs, err := libp2psetup.MakePubsub(ctx, newHost)
	if err != nil {
		t.Fatalf("error making pubsub: %v", err)
	}
	newTopic, err := newPs.Join(topicString)
	if err != nil {
		t.Fatalf("error joining topic: %v", err)
	}
	defer newTopic.Close()
	newNode := NewRaftNode(newHost, newPs, newTopic, 100, 500*time.Millisecond)
	go newNode.Run(ctx)

	time.Sleep(1 * time.Second)
	leaderIds = make(map[string]any)
	terms = make(map[int]any)

	leaderIds[newNode.GetLeader()] = struct{}{}
	terms[newNode.GetCurrentTerm()] = struct{}{}
	for i := range testItems.RaftNodes {
		leader := testItems.RaftNodes[i].GetLeader()
		term := testItems.RaftNodes[i].GetCurrentTerm()
		leaderIds[leader] = struct{}{}
		terms[term] = struct{}{}
	}
	assert.Equal(t, 1, len(leaderIds))
	assert.Equal(t, 1, len(terms))

	// If leader disconnects, a leader should be elected
	var prevLeaderID string
	for i := range testItems.RaftNodes {
		if testItems.RaftNodes[i].GetRole() == Leader {
			prevLeaderID = testItems.RaftNodes[i].GetHostId()
			testItems.Hosts[i].Close()
			break
		}
	}

	time.Sleep(1 * time.Second)
	var newLeaderID string
	for i := range testItems.RaftNodes {
		if testItems.RaftNodes[i].GetRole() == Leader {
			newLeaderID = testItems.RaftNodes[i].GetHostId()
			break
		}
	}
	assert.NotEqual(t, prevLeaderID, newLeaderID)
}
