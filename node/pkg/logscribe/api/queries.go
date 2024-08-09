package api

const (
	ReadLogs = `SELECT * FROM logs LIMIT @limit OFFSET @offset;`
)
