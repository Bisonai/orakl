-- Intentionally non-destructive: keep signer_key on rollback so the durable keyring (the active
-- key plus any in-flight pending keys) survives a downgrade and a subsequent re-upgrade. Pre-fix
-- binaries ignore this additive table, so retaining it is safe, and recovery after an interrupted
-- rotation depends on not losing these locally-held keys (issue #2516). A subsequent up-migration
-- is a no-op (CREATE TABLE IF NOT EXISTS + seed WHERE NOT EXISTS), preserving the rows.
-- To fully remove the table, drop it manually: DROP TABLE IF EXISTS signer_key;
SELECT 1;
