CREATE TABLE IF NOT EXISTS logs (
    id SERIAL PRIMARY KEY,
    service TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    level INT4 NOT NULL,
    message TEXT NOT NULL,
    fields JSONB NOT NULL
);

CREATE INDEX idx_logs_timestamp ON logs (timestamp);
CREATE INDEX idx_logs_level ON logs (level);
CREATE INDEX idx_logs_service ON logs (service);