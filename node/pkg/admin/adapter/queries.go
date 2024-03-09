package adapter

const (
	InsertAdapter = `INSERT INTO adapters (name) VALUES (@name) RETURNING *;`

	UpsertAdapter = `
	INSERT INTO adapters (name) VALUES (@name)
	ON CONFLICT (name) DO UPDATE SET active = true
	RETURNING *;`

	InsertFeed = `INSERT INTO feeds (name, definition, adapter_id) VALUES (@name, @definition, @adapter_id) RETURNING *;`

	UpsertFeed = `
	INSERT INTO feeds (name, definition, adapter_id)
    VALUES (@name, @definition, @adapter_id)
    ON CONFLICT (name) DO UPDATE SET
    definition = EXCLUDED.definition,
    adapter_id = EXCLUDED.adapter_id
    RETURNING *;`

	GetAdapter = `SELECT * FROM adapters;`

	GetAdapterByName = `SELECT * FROM adapters WHERE name = @name LIMIT 1;`

	GetAdapterById = `SELECT * FROM adapters WHERE id = @id;`

	GetFeedsByAdapterId = `SELECT * FROM feeds WHERE adapter_id = @id;`

	DeleteAdapterById = `DELETE FROM adapters WHERE id = @id RETURNING *;`

	ActivateAdapter = `UPDATE adapters SET active = true WHERE id = @id RETURNING *;`

	DeactivateAdapter = `UPDATE adapters SET active = false WHERE id = @id RETURNING *;`
)
