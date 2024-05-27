DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'submission_addresses'
        AND column_name = 'interval'
    ) THEN
        ALTER TABLE submission_addresses
        DROP COLUMN interval;
    END IF;
END $$;