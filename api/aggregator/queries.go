package aggregator

import (
	"strconv"
	"strings"
)

type GetAggregatorQueryParams struct {
	Active  string
	Chain   string
	Address string
}

const (
	InsertAggregator = `
		INSERT INTO aggregators (
			aggregator_hash,
			active,
			name,
			address,
			heartbeat,
			threshold,
			absolute_threshold,
			adapter_id,
			chain_id,
			fetcher_type
		) VALUES (
			@aggregator_hash,
			@active,
			@name,
			@address,
			@heartbeat,
			@threshold,
			@absolute_threshold,
			@adapter_id,
			@chain_id,
			@fetcher_type
		)
		RETURNING aggregator_id;
	`

	GetAggregatorByChainAndHash = `
		SELECT *
		FROM aggregators
		WHERE
			aggregator_hash = @aggregator_hash AND
			chain_id = @chain_id
		LIMIT 1;
	`

	RemoveAggregator       = `DELETE FROM aggregators WHERE aggregator_id = @id RETURNING *;`
	UpdateAggregator       = `UPDATE aggregators SET active = @active WHERE aggregator_id = @id AND chain_id = @chain_id RETURNING *;`
	UpdateAggregatorByHash = `UPDATE aggregators SET active = @active WHERE aggregator_hash = @hash AND chain_id = @chain_id RETURNING *;`
)

func GenerateGetAggregatorQuery(params GetAggregatorQueryParams) (string, error) {
	baseQuery := `SELECT * FROM aggregators`
	var conditionQueries []string
	if params.Active != "" {
		_, err := strconv.ParseBool(params.Active)
		if err != nil {
			return "", err
		}
		conditionQueries = append(conditionQueries, "active = "+params.Active)
	}
	if params.Chain != "" {
		conditionQueries = append(conditionQueries, "chain_id = (SELECT chain_id FROM chains WHERE name = '"+params.Chain+"')")
	}
	if params.Address != "" {
		conditionQueries = append(conditionQueries, "address = '"+params.Address+"'")
	}
	if len(conditionQueries) == 0 {
		return baseQuery, nil
	}
	joinedString := strings.Join(conditionQueries, " AND ")
	return baseQuery + " WHERE " + joinedString, nil
}
