CREATE TABLE IF NOT EXISTS proofs (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    round INT8 NOT NULL,
    proof BYTEA,
)