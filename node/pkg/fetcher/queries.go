package fetcher

const (
	loadActiveAdaptersQuery   = `SELECT * FROM adapters WHERE active = true`
	loadFeedsByAdapterIdQuery = `SELECT * FROM feeds WHERE adapter_id = @adapterId`
)
