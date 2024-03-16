package peer

const (
	InsertPeer = `INSERT INTO peers (ip, port, lib_id) VALUES (@ip, @port, @lib_id) RETURNING *;`

	UpsertPeer = `
		INSERT INTO peers (ip, port, lib_id) VALUES (@ip, @port, @lib_id)
		ON CONFLICT (ip) DO UPDATE SET port = @port, lib_id = @lib_id RETURNING *;
	`

	GetPeer = `SELECT * FROM peers;`

	DeletePeerById = `DELETE FROM peers WHERE id = @id RETURNING *;`
)
