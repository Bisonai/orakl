package contract

const (
	InsertContract = `
	INSERT INTO contracts (address) VALUES (@address) RETURNING *;
	`

	GetContract = `SELECT * FROM contracts;`

	GetConnectedReporters = `SELECT * FROM reporters WHERE id IN (SELECT "B" FROM "_ContractToReporter" WHERE "A" = @contractId);`

	GetConnectedFunctions = `SELECT * FROM functions WHERE contract_id = @contractId;`

	GetContractById = `SELECT * FROM contracts WHERE contract_id = @id;`

	ConnectReporter = `INSERT INTO "_ContractToReporter" ("A", "B") VALUES (@contractId, @reporterId);`

	DisconnectReporter = `DELETE FROM "_ContractToReporter" WHERE "A" = @contractId AND "B" = @reporterId;`

	UpdateContract = `UPDATE contracts SET address = @address WHERE contract_id = @id RETURNING *;`

	DeleteContract = `DELETE FROM contracts WHERE contract_id = @id RETURNING *;`
)
