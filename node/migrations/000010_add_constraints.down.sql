DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'global_aggregates_name_fkey'
    ) THEN
        ALTER TABLE global_aggregates
        DROP CONSTRAINT global_aggregates_name_fkey;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'local_aggregates_name_fkey'
    ) THEN
        ALTER TABLE local_aggregates
        DROP CONSTRAINT local_aggregates_name_fkey;
    END IF;
END $$;