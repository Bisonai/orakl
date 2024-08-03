package raft

func (r *Raft) GetRole() RoleType {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Role
}

func (r *Raft) GetCurrentTerm() int {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Term
}

func (r *Raft) IncreaseTerm() {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.Term++
}

func (r *Raft) GetLeader() string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.LeaderID
}

func (r *Raft) SubscribersCount() int {
	return len(r.Ps.ListPeers(r.Topic.String()))
}

func (r *Raft) GetHostId() string {
	return r.Host.ID().String()
}
