DO $$
BEGIN
    IF EXISTS(SELECT 1 FROM feeds) THEN
        DELETE FROM feeds;
    END IF;

    IF EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feeds' AND column_name = 'config_id') THEN
        ALTER TABLE feeds DROP COLUMN config_id;
    END IF;

    IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = 'feeds' AND column_name = 'adapter_id') THEN
        ALTER TABLE feeds ADD COLUMN adapter_id INT8 NOT NULL;
        ALTER TABLE feeds ADD CONSTRAINT feeds_adapter_id_fkey FOREIGN KEY(adapter_id) REFERENCES adapters(id) ON DELETE CASCADE;
    END IF;
END $$;