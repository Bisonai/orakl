DROP INDEX IF EXISTS feeds_name_config_id_key_idx;
CREATE UNIQUE INDEX feeds_name_key ON feeds (name);