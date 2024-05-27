DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'global_aggregates_name_fkey'
    ) THEN
        ALTER TABLE global_aggregates
        ADD CONSTRAINT global_aggregates_name_fkey
        FOREIGN KEY (name)
        REFERENCES aggregators (name)
        ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'local_aggregates_name_fkey'
    ) THEN
        ALTER TABLE local_aggregates
        ADD CONSTRAINT local_aggregates_name_fkey
        FOREIGN KEY (name)
        REFERENCES adapters (name)
        ON DELETE CASCADE;
    END IF;
END $$;