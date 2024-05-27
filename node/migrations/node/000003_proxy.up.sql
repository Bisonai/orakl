CREATE TABLE IF NOT EXISTS proxies (
    id SERIAL PRIMARY KEY NOT NULL,
    protocol TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    location TEXT,
    CONSTRAINT "proxies_protocol_host_port_key" UNIQUE ("protocol", "host", "port")
)
