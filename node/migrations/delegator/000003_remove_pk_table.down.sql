CREATE TABLE IF NOT EXISTS "fee_payers" (
    "privateKey" VARCHAR(66) NOT NULL,
    CONSTRAINT "fee_payers_privateKey_key" UNIQUE ("privateKey")
);