package l2aggregator

const (
	GetL2AggregatorPair = `
	SELECT * FROM l2aggregatorpair WHERE (l1_aggregator_address = @l1_aggregator_address AND chain_id = @chain_id) LIMIT 1;
	`
)
