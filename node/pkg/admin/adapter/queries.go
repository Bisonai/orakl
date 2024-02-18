package adapter

const (
	InsertAdapter = `INSERT INTO adapters (name) VALUES (@name) RETURNING *;`

	InsertFeed = `INSERT INTO feeds (name, definition, adapter_id) VALUES (@name, @definition, @adapter_id) RETURNING *;`

	GetAdapter = `SELECT * FROM adapters;`

	GetAdapterById = `SELECT * FROM adapters WHERE adapter_id = @id;`

	GetFeedsByAdapterId = `SELECT * FROM feeds WHERE adapter_id = @id;`

	DeleteAdapterById = `DELETE FROM adapters WHERE adapter_id = @id;`
)

// try later

// func generateInsertAdapterQuery(insertPayload AdapterInsertModel) string {
// 	feeds := make([]string, len(insertPayload.Feeds))
// 	for i, feed := range insertPayload.Feeds {
// 		feeds[i] = fmt.Sprintf("('%s', '%s', (SELECT id FROM new_adapter))", feed.Name, feed.Definition)
// 	}

// 	return fmt.Sprintf(`
// 		WITH new_adapter AS (
// 			INSERT INTO adapters (name) VALUES ('%s') RETURNING id
// 		),
// 		feeds_data(name, definition) AS (
// 			VALUES
// 				%s
// 		)
// 		INSERT INTO feeds (name, definition, adapter_id)
// 		SELECT name, definition, (SELECT id FROM new_adapter) FROM feeds_data RETURNING *;
// 	`, insertPayload.Name, strings.Join(feeds, ","))
// }
