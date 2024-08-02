package raft

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
