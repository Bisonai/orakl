// Package names holds node metric names shared with readers (sentinel) without
// registering the node's collectors in the reader's process.
package names

const (
	LocalAggregateValue      = "orakl_node_local_aggregate_value"
	LocalAggregateTimestamp  = "orakl_node_local_aggregate_timestamp_seconds"
	GlobalAggregateValue     = "orakl_node_global_aggregate_value"
	GlobalAggregateRound     = "orakl_node_global_aggregate_round"
	GlobalAggregateTimestamp = "orakl_node_global_aggregate_timestamp_seconds"

	FeedValue     = "orakl_node_feed_value"
	FeedTimestamp = "orakl_node_feed_timestamp_seconds"
	FeedVolume    = "orakl_node_feed_volume"

	P2PPublishedMessages = "orakl_node_p2p_published_messages_total"
	P2PPublishedBytes    = "orakl_node_p2p_published_bytes_total"
	P2PPeerConnects      = "orakl_node_p2p_peer_connects_total"
	P2PPeerDisconnects   = "orakl_node_p2p_peer_disconnects_total"

	RaftLeaderSilence    = "orakl_node_raft_leader_silence_seconds"
	RaftElectionsStarted = "orakl_node_raft_elections_started_total"

	NameLabel   = "name"
	ConfigLabel = "config"
	FeedLabel   = "feed"
	TypeLabel   = "type"
	ReasonLabel = "reason"
)
