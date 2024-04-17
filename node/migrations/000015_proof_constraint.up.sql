DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'proofs_name_round_proof_key'
    ) THEN
        ALTER TABLE proofs
        ADD CONSTRAINT proofs_name_round_proof_key UNIQUE (name, round, proof);
    END IF;
END $$;