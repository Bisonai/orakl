package raft

import "github.com/libp2p/go-libp2p/core/peer"

func (r *RaftNode) IncreaseTerm() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Term++
}

func (r *RaftNode) UpdateTerm(newTerm int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Term = newTerm
}

func (r *RaftNode) GetCurrentTerm() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Term
}

func (r *RaftNode) UpdateVoteReceived(votes int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.VotesReceived = votes
}

func (r *RaftNode) GetVoteReceived() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.VotesReceived
}

func (r *RaftNode) UpdateRole(role RoleType) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Role = role
}

func (r *RaftNode) GetRole() RoleType {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Role
}

func (r *RaftNode) GetVotedFor() string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.VotedFor
}

func (r *RaftNode) GetLeader() string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.LeaderID
}

func (r *RaftNode) UpdateLeader(leader string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.LeaderID = leader
}

func (r *RaftNode) UpdateVotedFor(votedFor string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.VotedFor = votedFor
}

func (r *RaftNode) SubscribersCount() int {
	return len(r.Subscribers())
}

func (r *RaftNode) Subscribers() []peer.ID {
	return r.Node.GetPubSub().ListPeers(r.Node.GetTopic().String())
}

func (r *RaftNode) GetHostId() string {
	return r.Node.GetHost().ID().String()
}
