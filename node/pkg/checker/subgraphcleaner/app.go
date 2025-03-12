package subgraphcleaner

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"bisonai.com/miko/node/pkg/db"
)

const (
	cleanupInterval   = 1 * time.Hour
	subgraphInfoQuery = `SELECT ds.id AS schema_id,
    ds.name AS schema_name,
    ds.subgraph,
    ds.version,
    s.name,
        CASE
            WHEN s.pending_version = v.id THEN 'pending'::text
            WHEN s.current_version = v.id THEN 'current'::text
            ELSE 'unused'::text
        END AS status,
    d.failed,
    d.synced
   FROM deployment_schemas ds,
    subgraphs.subgraph_deployment d,
    subgraphs.subgraph_version v,
    subgraphs.subgraph s
  WHERE d.deployment = ds.subgraph::text AND v.deployment = d.deployment AND v.subgraph = s.id AND (s.pending_version = v.id OR s.current_version = v.id)`

	getOffsetBlockNumberQuery = `WITH time_threshold AS (
    SELECT EXTRACT(EPOCH FROM (NOW() - INTERVAL '7 days')) AS threshold
)
SELECT
    block_number
FROM
    @schema.chain_event
WHERE
    time < (SELECT threshold FROM time_threshold)
ORDER BY
    block$ DESC
LIMIT 1;`

	cleanChainEventQuery       = "DELETE FROM @schema.chain_event WHERE block$ <= @block"
	cleanFeedUpdatedQuery      = "DELETE FROM @schema.feed_feed_updated WHERE block$ <= @block"
	cleanLatestChainEventQuery = "DELETE FROM @schema.latest_chain_event WHERE upper(block_range) <= @block"
)

type SubgraphInfo struct {
	SchemaId   int    `db:"schema_id"`
	SchemaName string `db:"schema_name"`
	Subgraph   string `db:"subgraph"`
	Version    int    `db:"version"`
	Name       string `db:"name"`
	Status     string `db:"status"`
	Failed     bool   `db:"failed"`
	Synced     bool   `db:"synced"`
}

type OffsetBlockNumber struct {
	BlockNumber int `db:"block$"`
}

func Start(ctx context.Context) error {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := run(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func run(ctx context.Context) error {
	subgraphs, err := loadSubgraphs(ctx)
	if err != nil {
		return err
	}

	for _, subgraphInfo := range subgraphs {

		switch {
		case strings.HasPrefix(subgraphInfo.Name, "Feed-"):
			schema := subgraphInfo.SchemaName
			offsetBlock, err := db.QueryRow[OffsetBlockNumber](ctx, getOffsetBlockNumberQuery, map[string]any{"schema": schema})
			if err != nil {
				continue
			}

			err = db.QueryWithoutResult(ctx, cleanChainEventQuery, map[string]any{"schema": schema, "block": offsetBlock})
			if err != nil {
				log.Error().Err(err).Str("schema", schema).Msg("failed to clear chain event table")
			}

			err = db.QueryWithoutResult(ctx, cleanFeedUpdatedQuery, map[string]any{"schema": schema, "block": offsetBlock})
			if err != nil {
				log.Error().Err(err).Str("schema", schema).Msg("failed to clear feed updated table")
			}

			err = db.QueryWithoutResult(ctx, cleanLatestChainEventQuery, map[string]any{"schema": schema, "block": offsetBlock})
			if err != nil {
				log.Error().Err(err).Str("schema", schema).Msg("failed to clear latest chain event table")
			}
		}

		time.Sleep(1 * time.Second) // avoid db stress
	}

	return nil
}

func loadSubgraphs(ctx context.Context) ([]SubgraphInfo, error) {
	return db.QueryRows[SubgraphInfo](ctx, subgraphInfoQuery, nil)
}
