DROP TABLE IF EXISTS rest_calls;
DROP TABLE IF EXISTS websocket_connections;
DROP TABLE IF EXISTS websocket_subscriptions;

DROP INDEX IF EXISTS websocket_subscriptions_connection_id_idx;
DROP INDEX IF EXISTS websocket_connections_api_key_idx;
DROP INDEX IF EXISTS rest_calls_api_key_idx;