package reporter

const (
	InsertReporter = `INSERT INTO reporters (address, organization_id) VALUES (@address, @organizationId) RETURNING *;`

	GetReporter = `SELECT * FROM reporters;`

	GetReporterById = `SELECT * FROM reporters WHERE id = @id;`

	GetConnectedContracts = `SELECT * FROM contracts WHERE contract_id IN (SELECT "A" FROM "_ContractToReporter" WHERE "B" = @reporterId);`

	GetOrganizationName = `SELECT name FROM organizations WHERE organization_id = @organizationId;`

	UpdateReporterById = `UPDATE reporters SET address = @address, organization_id = @organizationId WHERE id = @id RETURNING *;`

	DeleteReporterById = `DELETE FROM reporters WHERE id = @id RETURNING *;`
)
