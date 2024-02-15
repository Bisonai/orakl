CREATE TABLE IF NOT EXISTS adapters (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS feeds (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    definition JSONB NOT NULL,
    adapter_id INT16 NOT NULL,
    CONSTRAINT feeds_adapter_id_fkey
        FOREIGN KEY(adapter_id)
        REFERENCES adapters(id)
        ON DELETE CASCADE
);