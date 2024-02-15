package service

const (
	InsertService = `
	INSERT INTO services (name) VALUES (@name) RETURNING *;
	`

	GetService = `
	SELECT * FROM services;
	`

	GetServiceById = `
	SELECT * FROM services WHERE service_id = @id LIMIT 1;
	`

	GetServiceByName = `
	SELECT * FROM services WHERE name = @name LIMIT 1;
	`

	UpdateServiceById = `
	UPDATE services SET name = @name WHERE service_id = @id RETURNING *;
	`

	DeleteServiceById = `
	DELETE FROM services WHERE service_id = @id RETURNING *;
	`
)
