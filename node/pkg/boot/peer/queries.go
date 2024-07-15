package peer

const (
	InsertPeer = `INSERT INTO peers (url) VALUES (@url) ON CONFLICT (url) DO NOTHING RETURNING *;`

	GetPeer = `SELECT * FROM peers;`

	DeletePeerById = `DELETE FROM peers WHERE id = @id RETURNING *;`
)
