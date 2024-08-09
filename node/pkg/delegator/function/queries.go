package function

const (
	InsertFunction = `INSERT INTO functions (name, "encodedName", contract_id) VALUES (@name, @encodedName, @contract_id) RETURNING *;`

	GetFunction = `SELECT * FROM functions;`

	GetFunctionById = `SELECT * FROM functions WHERE id = @id;`

	GetContractById = `SELECT * FROM contracts WHERE contract_id = @id;`

	UpdateFunctionById = `UPDATE functions SET name = @name, "encodedName" = @encodedName, contract_id = @contract_id WHERE id = @id RETURNING *;`

	DeleteFunctionById = `DELETE FROM functions WHERE id = @id RETURNING *;`
)
