CREATE TABLE IF NOT EXISTS "observed_blocks" (
    block_key TEXT NOT NULL,
    block_number BIGINT NOT NULL,
    CONSTRAINT "observed_blocks_key" UNIQUE ("block_key")
)