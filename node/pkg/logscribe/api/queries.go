package api

const (
	ReadLogs   = `SELECT * FROM logs LIMIT @limit;`
	DeleteLogs = `DELETE FROM logs WHERE id IN (SELECT id FROM logs LIMIT @limit);`
)
