package config

const (
	InsertConfigQuery     = "INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval, decimals) VALUES (@name, @fetch_interval, @aggregate_interval, @submit_interval, @decimals) RETURNING *"
	SelectConfigQuery     = "SELECT * FROM configs"
	SelectConfigByIdQuery = "SELECT * FROM configs WHERE id = @id"
	DeleteConfigQuery     = "DELETE FROM configs WHERE id = @id RETURNING *"
	InsertFeedQuery       = "INSERT INTO feeds (name, definition, config_id) VALUES (@name, @definition, @config_id)"
	DeleteFeedQuery       = "DELETE FROM feeds WHERE id = @id RETURNING *"
)
