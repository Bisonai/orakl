CREATE TABLE IF NOT EXISTS feed_data (
    adapter_id INT8 NOT NULL,
    name TEXT NOT NULL,
    value INT8 NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT feed_data_adapter_id_fkey
        FOREIGN KEY(adapter_id)
        REFERENCES adapters(id)
        ON DELETE CASCADE
)