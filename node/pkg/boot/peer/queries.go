package peer

const (
	InsertPeer = `INSERT INTO peers (url) VALUES (@url) RETURNING *;`

	GetPeer = `SELECT * FROM peers;`

	DeletePeerById = `DELETE FROM peers WHERE id = @id RETURNING *;`
)
