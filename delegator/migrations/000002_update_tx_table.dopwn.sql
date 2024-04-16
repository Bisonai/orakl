DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'transactions'
        AND column_name = 'input'
        AND data_type = 'text'
    ) THEN
        ALTER TABLE transactions ALTER COLUMN input TYPE VARCHAR(1024);
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'transactions'
        AND column_name = 'rawTx'
        AND data_type = 'text'
    ) THEN
        ALTER TABLE transactions ALTER COLUMN "rawTx" TYPE VARCHAR(1024);
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'transactions'
        AND column_name = 'signedRawTx'
        AND data_type = 'text'
    ) THEN
        ALTER TABLE transactions ALTER COLUMN "signedRawTx" TYPE VARCHAR(1024);
    END IF;
END $$;