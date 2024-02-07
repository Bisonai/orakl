package feed

const (
	GetFeed = `SELECT * FROM feeds;`

	GetFeedById = `SELECT * FROM feeds WHERE feed_id = @id;`

	DeleteFeedById = `DELETE FROM feeds WHERE feed_id = @id RETURNING *`

	GetFeedsByAdapterId = `SELECT * FROM feeds WHERE adapter_id = @id;`
)
