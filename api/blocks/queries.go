package blocks

import (
	"fmt"
	"strings"
)

const (
	// get observedBlock given service
	GetObservedBlock = `
		SELECT * FROM observed_blocks
		WHERE service = @service;
	`

	// upsert to observed_blocks given service and block_number
	UpsertObservedBlock = `
		INSERT INTO observed_blocks (service, block_number)
		VALUES (@service, @block_number)
		ON CONFLICT (service) DO UPDATE SET block_number = GREATEST(observed_blocks.block_number, EXCLUDED.block_number)
		RETURNING *;
	`

	// get all unprocessed blocks given service
	GetUnprocessedBlocks = `
		SELECT * FROM unprocessed_blocks
		WHERE service = @service;
	`

	// delete unprocessed block given service and block_number
	DeleteUnprocessedBlock = `
		DELETE FROM unprocessed_blocks
		WHERE service = @service AND block_number = @block_number
		RETURNING *;
	`
)

func GenerateInsertBlocksQuery(blocks []int64, service string) string {
	baseQuery := `
		INSERT INTO unprocessed_blocks (service, block_number)
		VALUES
	`
	onConflict := `
		ON CONFLICT (service, block_number) DO NOTHING;
	`
	values := make([]string, 0, len(blocks))
	for _, block := range blocks {
		values = append(values, fmt.Sprintf("('%s', %d)", service, block))
	}

	return baseQuery + strings.Join(values, ",") + onConflict
}