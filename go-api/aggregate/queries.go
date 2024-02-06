package aggregate

const (
	InsertAggregate = `
	INSERT INTO aggregates (timestamp, value, aggregator_id) VALUES (@timestamp::timestamptz, @value, @aggregator_id) RETURNING aggregate_id;
	`

	GetAggregate = `
	SELECT * FROM aggregates;
	`

	GetAggregateById = `
	SELECT *
	FROM aggregates
	WHERE aggregate_id = @id
	LIMIT 1;
	`

	GetLatestAggregateByHash = `
	SELECT *
	FROM aggregates
	WHERE aggregator_id = (SELECT aggregator_id FROM aggregators WHERE aggregator_hash = @aggregator_hash)
	ORDER BY timestamp DESC
	LIMIT 1;
	`

	GetLatestAggregateById = `
	SELECT *
      FROM aggregates
      WHERE aggregator_id = @aggregator_id
      ORDER BY timestamp DESC
      LIMIT 1;
	`

	UpdateAggregateById = `
	UPDATE aggregates
		SET aggregator_id = @aggregator_id, timestamp = @timestamp::timestamptz, value = @value
		WHERE aggregate_id = @id
		RETURNING *;
	`

	DeleteAggregateById = `DELETE FROM aggregates WHERE aggregate_id = @id RETURNING *;`
)
