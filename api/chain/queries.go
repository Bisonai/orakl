package chain

const (
	GetChain = `SELECT * FROM chains;`

	GetChainByID = `SELECT * FROM chains WHERE chain_id = @id;`

	GetChainByName = `SELECT * FROM chains WHERE name = @name;`

	InsertChain = `INSERT INTO chains (name) VALUES (@name) RETURNING *;`

	UpdateChain = `UPDATE chains SET name = @name WHERE chain_id = @id RETURNING *;`

	RemoveChain = `DELETE FROM chains WHERE chain_id = @id RETURNING *;`
)
