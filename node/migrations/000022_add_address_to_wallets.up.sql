DO
$$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'wallets'
        AND column_name = 'address'
    ) THEN
        ALTER TABLE wallets ADD COLUMN address VARCHAR(42);
    END IF;
END
$$