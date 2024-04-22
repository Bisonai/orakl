DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'adapters'
        AND column_name = 'interval'
    ) THEN
        ALTER TABLE adapters
        ADD COLUMN interval INT4;
    END IF;
END $$;