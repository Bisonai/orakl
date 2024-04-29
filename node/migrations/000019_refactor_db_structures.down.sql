DO $$
BEGIN
    -- 1. Add back constraint columns

    -- feeds
    ALTER TABLE feeds DROP CONSTRAINT IF EXISTS feeds_config_id_fkey;
    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feeds' AND column_name = 'adapter_id') THEN
        ALTER TABLE feeds DROP COLUMN IF EXISTS config_id;
        ALTER TABLE feeds ADD COLUMN adapter_id INT8 NOT NULL;
    END IF;

    -- feed_data
    ALTER TABLE feed_data DROP CONSTRAINT IF EXISTS feed_data_config_id_fkey;
    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feed_data' AND column_name = 'name') THEN
        ALTER TABLE feed_data DROP COLUMN IF EXISTS feed_id;
        ALTER TABLE feed_data ADD COLUMN name TEXT NOT NULL;
    END IF;

    -- local_aggregates
    ALTER TABLE local_aggregates DROP CONSTRAINT IF EXISTS local_aggregates_config_id_fkey;
    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'local_aggregates' AND column_name = 'name') THEN
        ALTER TABLE local_aggregates DROP COLUMN IF EXISTS config_id;
        ALTER TABLE local_aggregates ADD COLUMN name TEXT NOT NULL;
    END IF;

    -- global_aggregates
    ALTER TABLE global_aggregates DROP CONSTRAINT IF EXISTS global_aggregates_config_id_fkey;
    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'global_aggregates' AND column_name = 'name') THEN
        ALTER TABLE global_aggregates DROP COLUMN IF EXISTS config_id;
        ALTER TABLE global_aggregates ADD COLUMN name TEXT NOT NULL;
    END IF;

    -- proofs
    ALTER TABLE proofs DROP CONSTRAINT IF EXISTS proofs_config_id_fkey;
    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'proofs' AND column_name = 'name') THEN
        ALTER TABLE proofs DROP COLUMN IF EXISTS config_id;
        ALTER TABLE proofs ADD COLUMN name TEXT NOT NULL;
    END IF;
    -- 2. Recreate dropped tables

    CREATE TABLE IF NOT EXISTS adapters (
    id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        active BOOLEAN NOT NULL DEFAULT TRUE,
        interval INTEGER NOT NULL DEFAULT 2000
    );

    CREATE TABLE IF NOT EXISTS aggregators (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        active BOOLEAN NOT NULL DEFAULT TRUE,
        interval INTEGER NOT NULL DEFAULT 5000
    );

    CREATE TABLE IF NOT EXISTS submission_addresses (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        address TEXT NOT NULL,
        interval INTEGER
    );

END $$;