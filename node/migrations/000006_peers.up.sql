CREATE TABLE IF NOT EXISTS peers (
    id SERIAL PRIMARY KEY,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    host_id TEXT NOT NULL UNIQUE
);