DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'transactions'
        AND column_name = 'input'
        AND data_type = 'character varying'
    ) THEN
        ALTER TABLE transactions ALTER COLUMN input TYPE TEXT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'transactions'
        AND column_name = 'rawTx'
        AND data_type = 'character varying'
    ) THEN
        ALTER TABLE transactions ALTER COLUMN "rawTx" TYPE TEXT;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'transactions'
        AND column_name = 'signedRawTx'
        AND data_type = 'character varying'
    ) THEN
        ALTER TABLE transactions ALTER COLUMN "signedRawTx" TYPE TEXT;
    END IF;
END $$;