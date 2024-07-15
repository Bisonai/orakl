package event

import (
	"fmt"
)

const (
	SubgraphInfoQuery = `SELECT ds.id AS schema_id,
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
  WHERE d.deployment = ds.subgraph::text AND v.deployment = d.deployment AND v.subgraph = s.id;`
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

type Config struct {
	Name           string `json:"name"`
	SubmitInterval int    `json:"submitInterval"`
}

type PegPorConfig struct {
	Name      string `json:"name"`
	Heartbeat int    `json:"heartbeat"`
}

type FeedToCheck struct {
	SchemaName          string
	FeedName            string
	ExpectedInterval    int
	LatencyChecked      int
	OversubmissionCount int
}

type FullfillEventToCheck struct {
	SchemaName string
	Name       string
	EventName  string
}

type CheckList struct {
	Feeds []FeedToCheck
	Por   FeedToCheck
	VRF   FullfillEventToCheck
}

func feedEventQuery(schemaName string) string {
	return fmt.Sprintf(`SELECT time FROM %s.feed_feed_updated ORDER BY time DESC LIMIT 1;`, schemaName)
}

func feedLastIntervalEventQuery(schemaName string, interval int) string {
	intervalInSeconds := interval / 1000
	return fmt.Sprintf(`SELECT COUNT(*) FROM %s.feed_feed_updated WHERE time >= EXTRACT(EPOCH FROM NOW()) - %d;`, schemaName, intervalInSeconds)
}

func aggregatorEventQuery(schemaName string) string {
	return fmt.Sprintf(`SELECT time FROM %s.aggregator_submission_received ORDER BY time DESC LIMIT 1;`, schemaName)
}

func loadUnfullfilledVRFEventQuery(schemaName string, eventName string) string {
	return fmt.Sprintf(`SELECT block$, id, time FROM %s.%s WHERE success = false ORDER BY time DESC;`, schemaName, eventName)
}

func loadOraklConfigUrl(chain string) string {
	return fmt.Sprintf("https://config.orakl.network/%s_configs.json", chain)
}

func loadPegPorConfigUrl(chain string) string {
	return fmt.Sprintf("https://config.orakl.network/aggregator/%s/peg.por.json", chain)
}
