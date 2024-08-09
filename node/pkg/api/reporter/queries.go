package reporter

import (
	"strings"
)

type GetReporterQueryParams struct {
	ChainId       string
	ServiceId     string
	OracleAddress string
}

const (
	InsertReporter = `
		INSERT INTO reporters (address, "privateKey", "oracleAddress", chain_id, service_id)
		VALUES (@address, @privateKey, @oracleAddress, @chain_id, @service_id)
		RETURNING reporters.reporter_id, reporters.address, "reporters"."privateKey", "reporters"."oracleAddress",
			(SELECT name FROM chains WHERE chains.chain_id = reporters.chain_id) AS chain_name,
			(SELECT name FROM services WHERE services.service_id = reporters.service_id) AS service_name;
			`

	GetReporterById = `
		SELECT reporters.reporter_id, reporters.address, "reporters"."privateKey", "reporters"."oracleAddress", chains.name AS chain_name, services.name AS service_name
		FROM reporters
		JOIN chains ON reporters.chain_id = chains.chain_id
		JOIN services ON reporters.service_id = services.service_id
		WHERE reporter_id = @id LIMIT 1;`

	UpdateReporterById = `
		UPDATE reporters
		SET address = @address, "privateKey" = @privateKey, "oracleAddress" = @oracleAddress
		WHERE reporter_id = @id
		RETURNING reporters.reporter_id, reporters.address, "reporters"."privateKey", "reporters"."oracleAddress",
		(SELECT name FROM chains WHERE chains.chain_id = reporters.chain_id) AS chain_name,
		(SELECT name FROM services WHERE services.service_id = reporters.service_id) AS service_name;
		`

	DeleteReporterById = `DELETE FROM reporters WHERE reporter_id = @id RETURNING reporters.reporter_id, reporters.address, "reporters"."privateKey", "reporters"."oracleAddress",
	(SELECT name FROM chains WHERE chains.chain_id = reporters.chain_id) AS chain_name,
	(SELECT name FROM services WHERE services.service_id = reporters.service_id) AS service_name;`
)

func GenerateGetReporterQuery(params GetReporterQueryParams) string {
	baseQuery := `
	SELECT reporters.reporter_id, reporters.address, "reporters"."privateKey", "reporters"."oracleAddress", chains.name AS chain_name, services.name AS service_name
	FROM reporters
	JOIN chains ON reporters.chain_id = chains.chain_id
	JOIN services ON reporters.service_id = services.service_id
	`
	var conditionQueries []string
	if params.ChainId != "" {
		conditionQueries = append(conditionQueries, "reporters.chain_id = "+params.ChainId)
	}
	if params.ServiceId != "" {
		conditionQueries = append(conditionQueries, "reporters.service_id = "+params.ServiceId)
	}
	if params.OracleAddress != "" {
		conditionQueries = append(conditionQueries, "\"reporters\".\"oracleAddress\" = '"+params.OracleAddress+"'")
	}
	if len(conditionQueries) == 0 {
		return baseQuery
	}
	joinedString := strings.Join(conditionQueries, " AND ")
	return baseQuery + " WHERE " + joinedString
}
