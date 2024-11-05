DROP INDEX IF EXISTS feeds_name_key;
CREATE UNIQUE INDEX feeds_name_config_id_key_idx ON feeds (name, config_id);