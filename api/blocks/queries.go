package blocks

const (
	// upsert to observed_blocks given service and block_number
	UpsertObservedBlock = `
		INSERT INTO observed_blocks (service_id, block_number)
		VALUES (@service, @block_number)
		ON CONFLICT (service) DO UPDATE SET block_number = @block_number
		RETURNING *;
	`

	// upsert to unprocessed_blocks given service and block_number
	UpsertUnprocessedBlock = `
		INSERT INTO unprocessed_blocks (service, block_number)
		VALUES (@service, @block_number)
		ON CONFLICT (service, block_number) DO NOTHING
		RETURNING *;
	`

	// get all unprocessed blocks given service
	GetUnprocessedBlocks = `
		SELECT * FROM unprocessed_blocks WHERE service = @service
	`

	// delete unprocessed block given service and block_number
	DeleteUnprocessedBlock = `
		DELETE FROM unprocessed_blocks WHERE service = @service AND block_number = @block_number
	`
)