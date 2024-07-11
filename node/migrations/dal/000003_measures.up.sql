CREATE TABLE IF NOT EXISTS rest_calls (
    id SERIAL PRIMARY KEY,
    api_key TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    timestamp TIMESTAMPTZ  DEFAULT CURRENT_TIMESTAMP NOT NULL,
    status_code INT NOT NULL,
    response_time INT NOT NULL
);

CREATE TABLE IF NOT EXISTS websocket_connections (
    id SERIAL PRIMARY KEY,
    api_key TEXT NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    connection_end TIMESTAMPTZ,
    duration INTERVAL
);

CREATE TABLE IF NOT EXISTS websocket_subscriptions (
    id SERIAL PRIMARY KEY,
    connection_id INT NOT NULL REFERENCES websocket_connections(id),
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    topic TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS rest_calls_api_key_idx ON rest_calls(api_key);
CREATE INDEX IF NOT EXISTS websocket_connections_api_key_idx ON websocket_connections(api_key);
CREATE INDEX IF NOT EXISTS websocket_subscriptions_connection_id_idx ON websocket_subscriptions(connection_id);