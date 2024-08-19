CREATE INDEX IF NOT EXISTS local_aggregates_config_id_idx ON local_aggregates(config_id);
CREATE INDEX IF NOT EXISTS local_aggregates_timestamp_idx ON local_aggregates(timestamp);

CREATE INDEX IF NOT EXISTS global_aggregates_config_id_idx ON global_aggregates(config_id);
CREATE INDEX IF NOT EXISTS global_aggregates_timestamp_idx ON global_aggregates(timestamp);