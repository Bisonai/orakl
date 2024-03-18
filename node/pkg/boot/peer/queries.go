package peer

const (
	InsertPeer = `INSERT INTO peers (ip, port, host_id) VALUES (@ip, @port, @host_id) RETURNING *;`

	UpsertPeer = `
		INSERT INTO peers (ip, port, host_id) VALUES (@ip, @port, @host_id)
		ON CONFLICT (ip) DO UPDATE SET port = @port, host_id = @host_id RETURNING *;
	`

	GetPeer = `SELECT * FROM peers;`

	DeletePeerById = `DELETE FROM peers WHERE id = @id RETURNING *;`
)
