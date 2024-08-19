CREATE INDEX IF NOT EXISTS rest_calls_timestamp_key_idx ON rest_calls(timestamp);
CREATE INDEX IF NOT EXISTS websocket_connections_timestamp_idx ON websocket_connections(timestamp);
CREATE INDEX IF NOT EXISTS websocket_subscriptions_connection_idx ON websocket_subscriptions(connection_id);