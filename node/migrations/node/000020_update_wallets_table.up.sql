DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'wallets'
        AND column_name = 'pk'
        AND data_type = 'character varying'
    ) THEN
        ALTER TABLE wallets ALTER COLUMN pk TYPE text;
    END IF;
END $$;