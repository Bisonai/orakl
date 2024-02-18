package feed

const (
	GetFeed = `SELECT * FROM feeds;`

	GetFeedById = `SELECT * FROM feeds WHERE id = @id;`

	GetFeedsByAdapterId = `SELECT * FROM feeds WHERE adapter_id = @adapter_id;`
)
