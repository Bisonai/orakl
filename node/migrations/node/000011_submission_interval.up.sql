DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'submission_addresses'
        AND column_name = 'interval'
    ) THEN
        ALTER TABLE submission_addresses
        ADD COLUMN interval int4;
    END IF;
END $$;