DO $$
BEGIN
    -- 1. remove constraint columns if exists

    -- feeds
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feeds' AND column_name = 'adapter_id') THEN
        IF EXISTS(SELECT 1 FROM feeds) THEN
            DELETE FROM feeds;
        END IF;
        ALTER TABLE feeds DROP COLUMN adapter_id;
        ALTER TABLE feeds ADD COLUMN config_id INT4 NOT NULL;
        ALTER TABLE feeds ADD CONSTRAINT feeds_config_id_fkey FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE;
    END IF;

    -- feed_data
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feed_data' AND column_name = 'name') THEN
        IF EXISTS(SELECT 1 FROM feed_data) THEN
            DELETE FROM feed_data;
        END IF;
        ALTER TABLE feed_data DROP COLUMN name;
        ALTER TABLE feed_data DROP COLUMN adapter_id;
        ALTER TABLE feed_data ADD COLUMN feed_id INT4 NOT NULL;
        ALTER TABLE feed_data ADD CONSTRAINT feed_data_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE;
    END IF;

    -- local_aggregates
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'local_aggregates' AND column_name = 'name') THEN
        IF EXISTS(SELECT 1 FROM local_aggregates) THEN
            DELETE FROM local_aggregates;
        END IF;
        ALTER TABLE local_aggregates DROP COLUMN name;
        ALTER TABLE local_aggregates ADD COLUMN config_id INT4 NOT NULL;
        ALTER TABLE local_aggregates ADD CONSTRAINT local_aggregates_config_id_fkey FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE;
    END IF;

    -- global_aggregates
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'global_aggregates' AND column_name = 'name') THEN
        IF EXISTS(SELECT 1 FROM global_aggregates) THEN
            DELETE FROM global_aggregates;
        END IF;
        ALTER TABLE global_aggregates DROP COLUMN name;
        ALTER TABLE global_aggregates ADD COLUMN config_id INT4 NOT NULL;
        ALTER TABLE global_aggregates ALTER COLUMN round TYPE INT4 USING round::integer;
        ALTER TABLE global_aggregates ADD CONSTRAINT global_aggregates_config_id_fkey FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE;
    END IF;

    -- proofs
    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'proofs' AND column_name = 'name') THEN
        IF EXISTS(SELECT 1 FROM proofs) THEN
            DELETE FROM proofs;
        END IF;
        ALTER TABLE proofs DROP COLUMN name;
        ALTER TABLE proofs ADD COLUMN config_id INT4 NOT NULL;
        ALTER TABLE proofs ALTER COLUMN round TYPE INT4 USING round::integer;
        ALTER TABLE proofs ADD CONSTRAINT proofs_config_id_fkey FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE;
        ALTER TABLE proofs ADD CONSTRAINT proofs_config_id_round_key UNIQUE (config_id, round);
    END IF;

    -- 2. drop tables

    DROP TABLE IF EXISTS adapters;
    DROP TABLE IF EXISTS aggregators;
    DROP TABLE IF EXISTS submission_addresses;
END $$;