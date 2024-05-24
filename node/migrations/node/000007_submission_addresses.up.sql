CREATE TABLE IF NOT EXISTS submission_addresses (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    address TEXT NOT NULL
)