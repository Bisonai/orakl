DELETE FROM peers;
ALTER TABLE peers ADD COLUMN IF NOT EXISTS url TEXT;
ALTER TABLE peers DROP COLUMN IF EXISTS ip, DROP COLUMN IF EXISTS port, DROP COLUMN IF EXISTS host_id;