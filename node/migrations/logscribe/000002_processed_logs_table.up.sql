CREATE TABLE IF NOT EXISTS processed_logs (
    id SERIAL PRIMARY KEY,
    log_hash TEXT NOT NULL
);

CREATE INDEX idx_processed_logs_log_hash ON processed_logs (log_hash);