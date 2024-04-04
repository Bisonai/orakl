package providerUrl

const (
	InsertProviderUrl     = `INSERT INTO provider_urls (chain, chain_id, url, priority) VALUES (@chain, @chain_id, @url, @priority) RETURNING *;`
	GetProviderUrl        = `SELECT * FROM provider_urls;`
	GetProviderUrlById    = `SELECT * FROM provider_urls WHERE id = @id;`
	DeleteProviderUrlById = `DELETE FROM provider_urls WHERE id = @id RETURNING *;`
)
