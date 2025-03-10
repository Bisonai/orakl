//nolint:all
package raft

import (
	"context"
	"fmt"
	"testing"
	"time"

	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
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

func WaitForCondition(ctx context.Context, t *testing.T, condition func() bool) {
	t.Helper()
	timer := time.NewTimer(time.Second * 60)
	defer timer.Stop()
	for {
		if condition() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled")
		case <-timer.C:
			t.Fatalf("wait for condition timed out")
		default:
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func allNodesHaveSameTerm(nodes []*Raft) bool {
	for i := 1; i < len(nodes); i++ {
		if nodes[0].GetCurrentTerm() < 1 || nodes[0].GetCurrentTerm() != nodes[i].GetCurrentTerm() {
			return false
		}
	}
	fmt.Println("all nodes have same term: ", nodes[0].GetCurrentTerm())
	return true
}

func allNodesHaveSameTermMin(nodes []*Raft, min int) bool {
	if len(nodes) == 1 {
		return nodes[0].GetCurrentTerm() > min
	}

	for i := 1; i < len(nodes); i++ {
		if nodes[0].GetCurrentTerm() <= min || nodes[0].GetCurrentTerm() != nodes[i].GetCurrentTerm() {
			return false
		}
	}
	fmt.Println("all nodes have same term: ", nodes[0].GetCurrentTerm())
	return true
}

func isolateNode(nodes []*Raft, node *Raft) {
	for _, entry := range nodes {
		if entry.GetHostId() == node.GetHostId() {
			continue
		}
		entry.addBlacklist(node.GetHostId())
	}
}

func recoverNode(nodes []*Raft, node *Raft) {
	for _, entry := range nodes {
		if entry.GetHostId() == node.GetHostId() {
			continue
		}
		entry.removeBlacklist(node.GetHostId())
	}
}

func setup(ctx context.Context, cancel context.CancelFunc, numOfNodes int) (func() error, *TestItems, error) {
	hosts := make([]host.Host, 0)
	for i := 0; i < numOfNodes; i++ {
		host, err := libp2psetup.NewHost(ctx)
		if err != nil {
			return nil, nil, err
		}
		hosts = append(hosts, host)
	}

	pss := make([]*pubsub.PubSub, 0)
	for i := 0; i < numOfNodes; i++ {
		ps, err := libp2psetup.MakePubsub(ctx, hosts[i])
		if err != nil {
			return nil, nil, err
		}
		pss = append(pss, ps)
	}

	for i := 0; i < numOfNodes; i++ {
		host := hosts[i]
		host.Connect(ctx, peer.AddrInfo{ID: hosts[(i+1)%numOfNodes].ID(), Addrs: hosts[(i+1)%numOfNodes].Addrs()})
	}

	topics := make([]*pubsub.Topic, 0)
	for i := 0; i < numOfNodes; i++ {
		topic, err := pss[i].Join(topicString)
		if err != nil {
			return nil, nil, err
		}
		topics = append(topics, topic)
	}

	raftNodes := []*Raft{}
	for i := 0; i < numOfNodes; i++ {
		node := NewRaftNode(hosts[i], pss[i], topics[i], 100, time.Second)
		node.LeaderJob = func(context.Context) error {
			log.Info().Str("HostId", node.Host.ID().ShortString()).Msg("Leader job")
			// log.Debug().Int("subscribers", node.SubscribersCount()).Int("Term", node.GetCurrentTerm()).Str("HostId", node.Host.ID().ShortString()).Msg("Leader job")
			// node.IncreaseTerm()
			return nil
		}
		node.HandleCustomMessage = func(ctx context.Context, message Message) error {
			log.Debug().Msg("Custom message")
			return fmt.Errorf("Unknown message type")
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

func setupRaftCluster(ctx context.Context, cancel context.CancelFunc, numOfNodes int, t *testing.T) (cleanup func() error, testItems *TestItems) {
	cleanup, testItems, err := setup(ctx, cancel, numOfNodes)
	if err != nil {
		cancel()
		t.Fatalf("error setting up test: %v", err)
	}
	return cleanup, testItems
}

func joinNewNode(ctx context.Context, testItems *TestItems) (*Raft, error) {
	newHost, err := libp2psetup.NewHost(ctx)
	if err != nil {
		return nil, err
	}

	newPs, err := libp2psetup.MakePubsub(ctx, newHost)
	if err != nil {
		return nil, err
	}

	for _, host := range testItems.Hosts {
		newHost.Connect(ctx, peer.AddrInfo{ID: host.ID(), Addrs: host.Addrs()})
	}

	newTopic, err := newPs.Join(topicString)
	if err != nil {
		return nil, err
	}

	newNode := NewRaftNode(newHost, newPs, newTopic, 100, time.Second)
	go newNode.Run(ctx)
	return newNode, nil
}

func TestRaft_LeaderElection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 3, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("Verify single leader across nodes", func(t *testing.T) {
		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTerm(testItems.RaftNodes)
		})

		leaderIds := make(map[string]struct{})
		for _, node := range testItems.RaftNodes {
			leader := node.GetLeader()
			leaderIds[leader] = struct{}{}
			assert.Equal(t, 2, node.SubscribersCount())
		}
		assert.Equal(t, 1, len(leaderIds))
	})

	t.Run("All nodes except leader should be followers", func(t *testing.T) {
		followerCount := 0
		for _, node := range testItems.RaftNodes {
			if node.GetRole() == Follower {
				followerCount++
			}
		}
		assert.Equal(t, len(testItems.RaftNodes)-1, followerCount)
	})

	t.Run("All nodes should have the same term", func(t *testing.T) {
		terms := make(map[int]struct{})
		for _, node := range testItems.RaftNodes {
			term := node.GetCurrentTerm()
			terms[term] = struct{}{}
		}
		assert.Equal(t, 1, len(terms))
	})
}

func TestRaft_LeaderResignAndReelection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 3, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("Leader resign and reelection", func(t *testing.T) {
		var lastTerm int
		for _, node := range testItems.RaftNodes {
			if node.GetRole() == Leader {
				lastTerm = node.GetCurrentTerm()
				node.ResignLeader()
				break
			}
		}
		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTermMin(testItems.RaftNodes, lastTerm)
		})

		leaderIds := make(map[string]struct{})
		for _, node := range testItems.RaftNodes {
			leader := node.GetLeader()
			leaderIds[leader] = struct{}{}
		}

		assert.Equal(t, 1, len(leaderIds))
	})
}

func TestRaft_NewNodeJoin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 3, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("New node join and expected behavior", func(t *testing.T) {
		newNode, err := joinNewNode(ctx, testItems)
		if err != nil {
			t.Fatalf("Failed to join new node: %v", err)
		}
		defer newNode.Host.Close()

		newList := []*Raft{newNode}
		newList = append(newList, testItems.RaftNodes...)

		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTerm(newList)
		})

		leaderIds := make(map[string]struct{})
		terms := make(map[int]struct{})

		leaderIds[newNode.GetLeader()] = struct{}{}
		terms[newNode.GetCurrentTerm()] = struct{}{}
		assert.Equal(t, 3, newNode.SubscribersCount())
		for _, node := range testItems.RaftNodes {
			leader := node.GetLeader()
			term := node.GetCurrentTerm()
			leaderIds[leader] = struct{}{}
			terms[term] = struct{}{}
			assert.Equal(t, 3, node.SubscribersCount())
		}
		assert.Equal(t, 1, len(leaderIds))
		assert.Equal(t, 1, len(terms))
	})
}

func TestRaft_LeaderDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 3, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("Leader disconnect and reelection", func(t *testing.T) {
		var prevLeaderID string
		var oldLeaderIndex int
		var lastTerm int
		for i, node := range testItems.RaftNodes {
			if node.GetRole() == Leader {
				oldLeaderIndex = i
				lastTerm = node.GetCurrentTerm()
				prevLeaderID = node.GetHostId()
				testItems.Hosts[i].Close()
				break
			}
		}
		assert.NotEmpty(t, prevLeaderID)

		newList := []*Raft{}
		for i, node := range testItems.RaftNodes {
			if i == oldLeaderIndex {
				continue
			}
			newList = append(newList, node)
		}

		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTermMin(newList, lastTerm)
		})

		var newLeaderID string
		for i, node := range testItems.RaftNodes {
			if i == oldLeaderIndex {
				continue
			}
			assert.Equal(t, 1, node.SubscribersCount())
			if node.GetRole() == Leader {
				newLeaderID = node.GetHostId()
				break
			}
		}

		assert.NotEmpty(t, newLeaderID)
		assert.NotEqual(t, prevLeaderID, newLeaderID)
	})
}

func TestRaft_NetworkPartitionAndRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 3, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	log.Info().Msg("Ready to test, setup 3 nodes")

	var previousLeaderNode *Raft
	var nodesExceptPreviousLeaderNode []*Raft
	var lastTerm int
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			previousLeaderNode = raftNode
			lastTerm = raftNode.GetCurrentTerm()
		} else {
			nodesExceptPreviousLeaderNode = append(nodesExceptPreviousLeaderNode, raftNode)
		}
	}

	assert.NotNil(t, previousLeaderNode)
	assert.NotEmpty(t, nodesExceptPreviousLeaderNode)

	// simulate partition, leader disconnect
	isolateNode(testItems.RaftNodes, previousLeaderNode)

	log.Info().Msg("Old leader is isolated")

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTermMin(nodesExceptPreviousLeaderNode, lastTerm)
	})

	var newLeaderNode *Raft
	for _, entry := range nodesExceptPreviousLeaderNode {
		if entry.GetRole() == Leader {
			newLeaderNode = entry
			break
		}
	}

	assert.NotNil(t, newLeaderNode)
	assert.NotEqual(t, previousLeaderNode.GetHostId(), newLeaderNode.GetHostId())

	log.Info().Msg("New leader is elected")

	recoverNode(testItems.RaftNodes, previousLeaderNode)

	log.Info().Msg("Old leader is recovered")

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	log.Info().Msg("New leader is elected after recovery")

	var newLeaderAfterRecovery *Raft
	leaderIds := make(map[string]struct{})
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			newLeaderAfterRecovery = raftNode
		}
		leaderIds[raftNode.GetLeader()] = struct{}{}
	}
	assert.Equal(t, 1, len(leaderIds))
	assert.Equal(t, newLeaderNode.GetHostId(), newLeaderAfterRecovery.GetHostId())
}

func TestRaft_MessageDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 3, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	log.Info().Msg("Ready to test, setup 3 nodes")

	leaderMap := make(map[string]struct{})
	var leader *Raft
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			leaderMap[raftNode.GetHostId()] = struct{}{}
			leader = raftNode
		}
	}
	assert.Equal(t, 1, len(leaderMap))
	assert.NotNil(t, leader)

	log.Info().Msg("Leader becomes slow")
	leader.setDelayedNode(true)

	time.Sleep(3 * time.Second)

	leaderMapAfter := make(map[string]struct{})
	var leaderAfter *Raft
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			leaderMapAfter[raftNode.GetHostId()] = struct{}{}
			leaderAfter = raftNode
		}
	}

	assert.Equal(t, 1, len(leaderMapAfter))
	assert.Equal(t, leader.GetHostId(), leaderAfter.GetHostId())

	log.Info().Msg("Leader recovers")
	leader.setDelayedNode(false)

	time.Sleep(3 * time.Second)

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	leaderMapAfterRecover := make(map[string]struct{})
	var leaderAfterRecover *Raft
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			leaderMapAfterRecover[raftNode.GetHostId()] = struct{}{}
			leaderAfterRecover = raftNode
		}
	}

	assert.Equal(t, 1, len(leaderMapAfterRecover))
	assert.Equal(t, leader.GetHostId(), leaderAfterRecover.GetHostId())
}

func TestRaft_LeaderElection_2_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 2, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("Verify single leader across nodes", func(t *testing.T) {
		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTerm(testItems.RaftNodes)
		})

		leaderIds := make(map[string]struct{})
		for _, node := range testItems.RaftNodes {
			leader := node.GetLeader()
			leaderIds[leader] = struct{}{}
			assert.Equal(t, 1, node.SubscribersCount())
		}
		assert.Equal(t, 1, len(leaderIds))
	})

	t.Run("All nodes except leader should be followers", func(t *testing.T) {
		followerCount := 0
		for _, node := range testItems.RaftNodes {
			if node.GetRole() == Follower {
				followerCount++
			}
		}
		assert.Equal(t, len(testItems.RaftNodes)-1, followerCount)
	})

	t.Run("All nodes should have the same term", func(t *testing.T) {
		terms := make(map[int]struct{})
		for _, node := range testItems.RaftNodes {
			term := node.GetCurrentTerm()
			terms[term] = struct{}{}
		}
		assert.Equal(t, 1, len(terms))
	})
}

func TestRaft_LeaderResignAndReelection_2_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 2, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("Leader resign and reelection", func(t *testing.T) {
		var lastTerm int
		for _, node := range testItems.RaftNodes {
			if node.GetRole() == Leader {
				lastTerm = node.GetCurrentTerm()
				node.ResignLeader()
				break
			}
		}
		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTermMin(testItems.RaftNodes, lastTerm)
		})

		leaderIds := make(map[string]struct{})
		for _, node := range testItems.RaftNodes {
			leader := node.GetLeader()
			leaderIds[leader] = struct{}{}
		}

		assert.Equal(t, 1, len(leaderIds))
	})
}

func TestRaft_NewNodeJoin_2_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 2, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("New node join and expected behavior", func(t *testing.T) {
		newNode, err := joinNewNode(ctx, testItems)
		if err != nil {
			t.Fatalf("Failed to join new node: %v", err)
		}
		defer newNode.Host.Close()

		newList := []*Raft{newNode}
		newList = append(newList, testItems.RaftNodes...)

		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTerm(newList)
		})

		leaderIds := make(map[string]struct{})
		terms := make(map[int]struct{})

		leaderIds[newNode.GetLeader()] = struct{}{}
		terms[newNode.GetCurrentTerm()] = struct{}{}
		assert.Equal(t, 2, newNode.SubscribersCount())
		for _, node := range testItems.RaftNodes {
			leader := node.GetLeader()
			term := node.GetCurrentTerm()
			leaderIds[leader] = struct{}{}
			terms[term] = struct{}{}
			assert.Equal(t, 2, node.SubscribersCount())
		}
		assert.Equal(t, 1, len(leaderIds))
		assert.Equal(t, 1, len(terms))
	})
}

func TestRaft_LeaderDisconnect_2_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 2, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	t.Run("Leader disconnect and reelection", func(t *testing.T) {
		var prevLeaderID string
		var oldLeaderIndex int
		var lastTerm int
		for i, node := range testItems.RaftNodes {
			if node.GetRole() == Leader {
				oldLeaderIndex = i
				lastTerm = node.GetCurrentTerm()
				prevLeaderID = node.GetHostId()
				testItems.Hosts[i].Close()
				break
			}
		}
		assert.NotEmpty(t, prevLeaderID)

		newList := []*Raft{}
		for i, node := range testItems.RaftNodes {
			if i == oldLeaderIndex {
				continue
			}
			newList = append(newList, node)
		}

		assert.Equal(t, 1, len(newList))

		WaitForCondition(ctx, t, func() bool {
			return allNodesHaveSameTermMin(newList, lastTerm)
		})

		var newLeaderID string
		for i, node := range testItems.RaftNodes {
			if i == oldLeaderIndex {
				continue
			}
			assert.Equal(t, 0, node.SubscribersCount())
			if node.GetRole() == Leader {
				newLeaderID = node.GetHostId()
				break
			}
		}

		assert.NotEmpty(t, newLeaderID)
		assert.NotEqual(t, prevLeaderID, newLeaderID)
	})
}

func TestRaft_NetworkPartitionAndRecovery_2_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 2, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	log.Info().Msg("Ready to test, setup 3 nodes")

	var previousLeaderNode *Raft
	var nodesExceptPreviousLeaderNode []*Raft
	var lastTerm int
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			previousLeaderNode = raftNode
			lastTerm = raftNode.GetCurrentTerm()
		} else {
			nodesExceptPreviousLeaderNode = append(nodesExceptPreviousLeaderNode, raftNode)
		}
	}

	assert.NotNil(t, previousLeaderNode)
	assert.NotEmpty(t, nodesExceptPreviousLeaderNode)

	// simulate partition, leader disconnect
	isolateNode(testItems.RaftNodes, previousLeaderNode)

	log.Info().Msg("Old leader is isolated")

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTermMin(nodesExceptPreviousLeaderNode, lastTerm)
	})

	var newLeaderNode *Raft
	for _, entry := range nodesExceptPreviousLeaderNode {
		if entry.GetRole() == Leader {
			newLeaderNode = entry
			break
		}
	}

	assert.NotNil(t, newLeaderNode)
	assert.NotEqual(t, previousLeaderNode.GetHostId(), newLeaderNode.GetHostId())

	log.Info().Msg("New leader is elected")

	recoverNode(testItems.RaftNodes, previousLeaderNode)

	log.Info().Msg("Old leader is recovered")

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	log.Info().Msg("New leader is elected after recovery")

	var newLeaderAfterRecovery *Raft
	leaderIds := make(map[string]struct{})
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			newLeaderAfterRecovery = raftNode
		}
		leaderIds[raftNode.GetLeader()] = struct{}{}
	}
	assert.Equal(t, 1, len(leaderIds))
	assert.Equal(t, newLeaderNode.GetHostId(), newLeaderAfterRecovery.GetHostId())
}

func TestRaft_MessageDelay_2_nodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, testItems := setupRaftCluster(ctx, cancel, 2, t)
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	for _, raftNode := range testItems.RaftNodes {
		go raftNode.Run(ctx)
	}

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	log.Info().Msg("Ready to test, setup 2 nodes")

	leaderMap := make(map[string]struct{})
	var leader *Raft
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			leaderMap[raftNode.GetHostId()] = struct{}{}
			leader = raftNode
		}
	}
	assert.Equal(t, 1, len(leaderMap))
	assert.NotNil(t, leader)

	log.Info().Msg("Leader becomes slow")
	leader.setDelayedNode(true)

	time.Sleep(3 * time.Second)

	leaderMapAfter := make(map[string]struct{})
	var leaderAfter *Raft
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			leaderMapAfter[raftNode.GetHostId()] = struct{}{}
			leaderAfter = raftNode
		}
	}

	assert.Equal(t, 1, len(leaderMapAfter))
	assert.Equal(t, leader.GetHostId(), leaderAfter.GetHostId())

	log.Info().Msg("Leader recovers")
	leader.setDelayedNode(false)

	time.Sleep(3 * time.Second)

	WaitForCondition(ctx, t, func() bool {
		return allNodesHaveSameTerm(testItems.RaftNodes)
	})

	leaderMapAfterRecover := make(map[string]struct{})
	var leaderAfterRecover *Raft
	for _, raftNode := range testItems.RaftNodes {
		if raftNode.GetRole() == Leader {
			leaderMapAfterRecover[raftNode.GetHostId()] = struct{}{}
			leaderAfterRecover = raftNode
		}
	}

	assert.Equal(t, 1, len(leaderMapAfterRecover))
	assert.Equal(t, leader.GetHostId(), leaderAfterRecover.GetHostId())
}
