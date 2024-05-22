DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'aggregators'
        AND column_name = 'interval'
    ) THEN
        ALTER TABLE aggregators
        DROP COLUMN interval;
    END IF;
END $$;