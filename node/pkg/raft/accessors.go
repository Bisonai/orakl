package raft

func (r *Raft) GetRole() RoleType {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	return r.Role
}

// IsLeader is GetRole() == Leader without taking Mutex, for callers outside
// the raft loop that must not wait on it (in-node reporter). Exact as long as
// Role is only changed through setRole.
func (r *Raft) IsLeader() bool {
	return r.leading.Load()
}

// setRole sets Role and its IsLeader mirror. The caller holds Mutex.
func (r *Raft) setRole(role RoleType) {
	r.Role = role
	r.leading.Store(role == Leader)
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
