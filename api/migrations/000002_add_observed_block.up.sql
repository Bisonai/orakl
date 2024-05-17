CREATE TABLE IF NOT EXISTS "observed_block" (
    block_key TEXT NOT NULL,
    block_number BIGINT NOT NULL,
    CONSTRAINT "observed_block_key" UNIQUE ("block_key")
)