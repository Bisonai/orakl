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

	NameLabel   = "name"
	ConfigLabel = "config"
	FeedLabel   = "feed"
)
