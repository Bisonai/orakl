package config

const (
	InsertConfigQuery     = "INSERT INTO configs (name, fetch_interval, aggregate_interval, submit_interval) VALUES (@name, @fetch_interval, @aggregate_interval, @submit_interval) RETURNING *"
	SelectConfigQuery     = "SELECT * FROM configs"
	SelectConfigByIdQuery = "SELECT * FROM configs WHERE id = @id"
	DeleteConfigQuery     = "DELETE FROM configs WHERE id = @id RETURNING *"
	BulkDeleteConfigQuery = "DELETE FROM configs WHERE id IN (@ids)"
	InsertFeedQuery       = "INSERT INTO feeds (name, definition, config_id) VALUES (@name, @definition, @config_id)"
	DeleteFeedQuery       = "DELETE FROM feeds WHERE id = @id RETURNING *"
	BulkDeleteFeedQuery   = "DELETE FROM feeds WHERE id IN (@ids)"
)
