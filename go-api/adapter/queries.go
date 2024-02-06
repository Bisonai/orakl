package adapter

const (
	InsertAdapter = `
	INSERT INTO adapters (adapter_hash, name, decimals) VALUES (@adapter_hash, @name, @decimals) RETURNING adapter_id;
	`

	InsertFeed = `
	INSERT INTO feeds (name, definition, adapter_id) VALUES (@name, @definition, @adapter_id) RETURNING feed_id;
	`

	GetAdapter = `SELECT * FROM adapters;`

	GetAdpaterById = `SELECT * FROM adapters WHERE adapter_id = @id;`

	GetAdapterByHash = `SELECT * FROM adapters WHERE adapter_hash = @adapter_hash`

	RemoveAdapter = `DELETE FROM adapters WHERE adapter_id = @id RETURNING *;`
)
