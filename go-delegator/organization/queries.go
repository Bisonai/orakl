package organization

const (
	InsertOrganization  = `INSERT INTO organizations (name) VALUES (@name) RETURNING *;`
	GetOrganization     = `SELECT * FROM organizations;`
	GetOrganizationById = `SELECT * FROM organizations WHERE organization_id = @id;`
	UpdateOrganization  = `UPDATE organizations SET name = @name WHERE organization_id = @id RETURNING *;`
	DeleteOrganization  = `DELETE FROM organizations WHERE organization_id = @id RETURNING *;`
)
