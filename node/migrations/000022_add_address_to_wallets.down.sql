DO
$$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'wallets'
        AND column_name = 'address'
    ) THEN
        ALTER TABLE wallets DROP COLUMN address
    END IF;
END
$$