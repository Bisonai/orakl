package feed

const (
	GetFeed = `SELECT * FROM feeds;`

	GetFeedById = `SELECT * FROM feeds WHERE id = @id;`

	GetFeedsByConfigId = `SELECT * FROM feeds WHERE config_id = @config_id;`
)
