package config

const (
	InsertConfigQuery     = "INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval, decimals, feed_data_freshness) VALUES (@name, @fetch_interval, @aggregate_interval, @submit_interval, @decimals, @feed_data_freshness) RETURNING *"
	SelectConfigQuery     = "SELECT id, name, fetch_interval, aggregate_interval, submit_interval, decimals, feed_data_freshness FROM configs"
	SelectConfigByIdQuery = "SELECT id, name, fetch_interval, aggregate_interval, submit_interval, decimals, feed_data_freshness FROM configs WHERE id = @id"
	DeleteConfigQuery     = "DELETE FROM configs WHERE id = @id RETURNING *"
	InsertFeedQuery       = "INSERT INTO feeds (name, definition, config_id) VALUES (@name, @definition, @config_id)"
	DeleteFeedQuery       = "DELETE FROM feeds WHERE id = @id RETURNING *"
)
