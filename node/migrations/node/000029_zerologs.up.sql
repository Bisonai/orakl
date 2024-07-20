CREATE TABLE IF NOT EXISTS zerologs (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    level INT4 NOT NULL,
    message TEXT NOT NULL,
    fields JSONB NOT NULL
);

CREATE INDEX idx_zerologs_timestamp ON zerologs (timestamp);
CREATE INDEX idx_zerologs_level ON zerologs (level);