package submissionAddress

const (
	InsertSubmissionAddress = `INSERT INTO submission_addresses (name, address) VALUES (@name, @address) RETURNING *;`

	UpsertSubmissionAddress = `INSERT INTO submission_addresses (name, address) VALUES (@name, @address) ON CONFLICT (name) DO UPDATE SET address = @address RETURNING *;`

	GetAggregatorNames = `SELECT name FROM aggregators WHERE active = true;`

	GetSubmissionAddress = `SELECT * FROM submission_addresses;`

	GetSubmissionAddressById = `SELECT * FROM submission_addresses WHERE id = @id;`

	DeleteSubmissionAddressById = `DELETE FROM submission_addresses WHERE id = @id RETURNING *;`

	UpdateSubmissionAddressById = `UPDATE submission_addresses SET name = @name, address = @address WHERE id = @id RETURNING *;`
)
