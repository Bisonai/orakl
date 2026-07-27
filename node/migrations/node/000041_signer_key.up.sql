-- Additive migration for crash-safe signer-key rotation (issue #2516).
-- The legacy `signer` table, its single row, and its `unique_dummy` constraint are
-- intentionally left in place: the pre-fix binary and the bootstrap/rollback path still
-- read/write it via LOAD_SIGNER / STORE_SIGNER. This migration only ADDS a new table.
CREATE TABLE IF NOT EXISTS signer_key (
    id          SERIAL PRIMARY KEY,
    address     TEXT,                            -- lowercased 0x address; NULL until the app backfills (pk is encrypted, cannot derive in SQL)
    pk          TEXT NOT NULL,                   -- encrypted with the same encryptor as the legacy signer.pk
    state       TEXT NOT NULL DEFAULT 'pending', -- 'active' | 'pending' | 'retired'
    tx_hash     TEXT,                            -- updateOracle tx hash for this candidate (advisory, nullable)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- At most one active row. Deliberately NO unique index on state='pending'
-- (a one-pending index would wedge rotation forever if a pending never lands).
CREATE UNIQUE INDEX IF NOT EXISTS signer_key_one_active ON signer_key ((state)) WHERE state = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS signer_key_addr       ON signer_key (address)  WHERE address IS NOT NULL;

-- Seed exactly once from the existing single `signer` row as the active key. This is only a
-- bootstrap: at runtime reconcile (loadCandidates) always also considers the current legacy
-- `signer` row as a candidate and, if it is the on-chain oracle, imports it into signer_key via
-- ensureActive. So a key a pre-fix binary writes to `signer` later is still discovered/adopted —
-- the one-time seed does not need to catch it. address stays NULL here (partial index tolerates
-- NULL); the app backfills it on first reconcile.
INSERT INTO signer_key (pk, state)
    SELECT pk, 'active' FROM signer
    WHERE pk IS NOT NULL AND pk <> '' AND NOT EXISTS (SELECT 1 FROM signer_key);
