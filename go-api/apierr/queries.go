package apierr

const (
	InsertError = `INSERT INTO error (request_id, timestamp, code, name, stack) VALUES (@request_id, @timestamp::timestamptz, @code, @name, @stack) RETURNING *`

	GetError = `SELECT * FROM error;`

	GetErrorById = `SELECT * FROM error WHERE error_id = @id`

	RemoveErrorById = `DELETE FROM error WHERE error_id = @id RETURNING *;`
)
