DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM feeds) THEN
        DELETE FROM feeds;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feeds' AND column_name = 'adapter_id') THEN
        ALTER TABLE feeds DROP COLUMN adapter_id;
    END IF;

    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feeds' AND column_name = 'config_id') THEN
        ALTER TABLE feeds ADD COLUMN config_id INT8 NOT NULL;
        ALTER TABLE feeds ADD CONSTRAINT feeds_config_id_fkey FOREIGN KEY(config_id) REFERENCES configs(id) ON DELETE CASCADE;
    END IF;
END $$;