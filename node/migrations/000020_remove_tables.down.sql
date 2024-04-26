CREATE TABLE IF NOT EXISTS adapters (
    id SERIAL PRIMARY KEY,
    name text NOT NULL,
    active bool NOT NULL DEFAULT true,
    interval int4 NOT NULL DEFAULT 2000,
);

CREATE TABLE IF NOT EXISTS aggregators (
    id SERIAL PRIMARY KEY,
    name test NOT NULL,
    active bool NOT NULL DEFAULT true,
    interval int4 NOT NULL DEFAULT 5000,
)

CREATE TABLE IF NOT EXISTS submission_addresses (
    id SERIAL PRIMARY KEY,
    name text NOT NULL,
    address text NOT NULL,
    interval int4
)