-- Step 1: Revert column type
ALTER TABLE rest_calls
ALTER COLUMN id TYPE INTEGER;

-- Step 2: Revert sequence type
ALTER SEQUENCE rest_calls_id_seq AS INTEGER;