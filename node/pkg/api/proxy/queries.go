package proxy

const (
	InsertProxy = `INSERT INTO proxies (protocol, host, port, location) VALUES (@protocol, @host, @port, @location) RETURNING *;`

	GetProxy = `SELECT * FROM proxies ORDER BY id asc;`

	GetProxyById = `SELECT * FROM proxies WHERE id = @id;`

	UpdateProxyById = `
	UPDATE proxies
	SET protocol = @protocol, host = @host, port = @port, location = @location
	WHERE id = @id 
	RETURNING *;
	`

	DeleteProxyById = `
	DELETE FROM proxies WHERE id = @id RETURNING *;
	`
)
