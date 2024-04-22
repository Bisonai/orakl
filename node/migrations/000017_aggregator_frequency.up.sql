DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'aggregators'
        AND column_name = 'interval'
    ) THEN
        ALTER TABLE aggregators
        ADD COLUMN interval INT4 DEFAULT 5000 NOT NULL;
    END IF;
END $$;