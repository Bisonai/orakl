package aggregator

const (
	InsertAggregator = `INSERT INTO aggregators (name) VALUES (@name) RETURNING *;`

	GetAggregator = `SELECT * FROM aggregators;`

	GetAggregatorById = `SELECT * FROM aggregators WHERE id = @id;`

	DeleteAggregatorById = `DELETE FROM aggregators WHERE id = @id RETURNING *;`

	ActivateAggregator = `UPDATE aggregators SET active = true WHERE id = @id RETURNING *;`

	DeactivateAggregator = `UPDATE aggregators SET active = false WHERE id = @id RETURNING *;`

	SyncAggregator = `INSERT INTO aggregators (name)
						SELECT name FROM adapters
						WHERE NOT EXISTS (
							SELECT 1 FROM aggregators WHERE aggregators.name = adapters.name
						) RETURNING *;`
)
