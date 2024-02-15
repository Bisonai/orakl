package listener

import (
	"strings"
)

type GetListenerQueryParams struct {
	ChainId   string
	ServiceId string
}

const (
	InsertListener = `
		INSERT INTO listeners (address, event_name, chain_id, service_id)
		VALUES (@address, @event_name, @chain_id, @service_id)
		RETURNING listeners.listener_id, listeners.address, listeners.event_name,
			(SELECT name from chains WHERE chain_id = listeners.chain_id) AS chain_name,
			(SELECT name from services WHERE service_id = listeners.service_id) AS service_name;
		`

	GetListenerById = `
		SELECT listeners.listener_id, listeners.address, listeners.event_name, chains.name AS chain_name, services.name AS service_name
		FROM listeners
		JOIN chains ON listeners.chain_id = chains.chain_id
		JOIN services ON listeners.service_id = services.service_id
		WHERE listener_id = @id
		LIMIT 1;
		`

	UpdateListenerById = `
		UPDATE listeners
		SET address = @address, event_name = @event_name
		WHERE listener_id = @id
		RETURNING listeners.listener_id, listeners.address, listeners.event_name,
			(SELECT name from chains WHERE chain_id = listeners.chain_id) AS chain_name,
			(SELECT name from services WHERE service_id = listeners.service_id) AS service_name;
		`

	DeleteListenerById = `
		DELETE FROM listeners WHERE listener_id = @id
		RETURNING listeners.listener_id, listeners.address, listeners.event_name,
		(SELECT name from chains WHERE chain_id = listeners.chain_id) AS chain_name,
		(SELECT name from services WHERE service_id = listeners.service_id) AS service_name;
		`
)

func GenerateGetListenerQuery(params GetListenerQueryParams) string {
	baseQuery := `
	SELECT listeners.listener_id, listeners.address, listeners.event_name, chains.name AS chain_name, services.name AS service_name
	FROM listeners
	JOIN chains ON listeners.chain_id = chains.chain_id
	JOIN services ON listeners.service_id = services.service_id
	`
	var conditionQueries []string
	if params.ChainId != "" {
		conditionQueries = append(conditionQueries, "listeners.chain_id = "+params.ChainId)
	}
	if params.ServiceId != "" {
		conditionQueries = append(conditionQueries, "listeners.service_id = "+params.ServiceId)
	}
	if len(conditionQueries) == 0 {
		return baseQuery
	}
	joinedString := strings.Join(conditionQueries, " AND ")
	return baseQuery + " WHERE " + joinedString
}
