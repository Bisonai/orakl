CREATE TABLE IF NOT EXISTS "observed_blocks" (
    service TEXT NOT NULL UNIQUE,
    block_number BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS "unprocessed_blocks" (
    service TEXT NOT NULL,
    block_number BIGINT NOT NULL,
    UNIQUE (service, block_number)
);