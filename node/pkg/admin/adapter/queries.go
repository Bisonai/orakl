package adapter

const (
	InsertAdapter = `INSERT INTO adapters (name) VALUES (@name) RETURNING *;`

	InsertFeed = `INSERT INTO feeds (name, definition, adapter_id) VALUES (@name, @definition, @adapter_id) RETURNING *;`

	GetAdapter = `SELECT * FROM adapters;`

	GetAdapterById = `SELECT * FROM adapters WHERE id = @id;`

	GetFeedsByAdapterId = `SELECT * FROM feeds WHERE adapter_id = @id;`

	DeleteAdapterById = `DELETE FROM adapters WHERE id = @id RETURNING *;`
)
