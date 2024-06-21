package raft

func (r *Raft) IncreaseTerm() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Term++
}

func (r *Raft) UpdateTerm(newTerm int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Term = newTerm
}

func (r *Raft) GetCurrentTerm() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Term
}

func (r *Raft) IncreaseVote() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.VotesReceived++
}

func (r *Raft) UpdateVoteReceived(votes int) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.VotesReceived = votes
}

func (r *Raft) GetVoteReceived() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.VotesReceived
}

func (r *Raft) UpdateRole(role RoleType) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Role = role
}

func (r *Raft) GetRole() RoleType {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Role
}

func (r *Raft) GetVotedFor() string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.VotedFor
}

func (r *Raft) GetLeader() string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.LeaderID
}

func (r *Raft) UpdateLeader(leader string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.LeaderID = leader
}

func (r *Raft) UpdateVotedFor(votedFor string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.VotedFor = votedFor
}

func (r *Raft) SubscribersCount() int {
	return len(r.Ps.ListPeers(r.Topic.String()))
}

func (r *Raft) GetHostId() string {
	return r.Host.ID().String()
}
