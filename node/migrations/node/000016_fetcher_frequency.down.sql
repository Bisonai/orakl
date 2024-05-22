DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'adapters'
        AND column_name = 'interval'
    ) THEN
        ALTER TABLE adapters
        DROP COLUMN interval;
    END IF;
END $$;