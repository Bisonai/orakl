-- CreateTable
CREATE TABLE IF NOT EXISTS "organizations" (
    "organization_id" BIGSERIAL NOT NULL,
    "name" VARCHAR(50) NOT NULL,
    CONSTRAINT "organizations_pkey" PRIMARY KEY ("organization_id"),
    CONSTRAINT "organizations_name_key" UNIQUE ("name")
);

-- CreateTable
CREATE TABLE IF NOT EXISTS "reporters" (
    "id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "organization_id" BIGINT NOT NULL,
    CONSTRAINT "reporters_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "reporters_address_key" UNIQUE ("address")
);

-- CreateTable
CREATE TABLE IF NOT EXISTS "contracts" (
    "contract_id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    CONSTRAINT "contracts_pkey" PRIMARY KEY ("contract_id"),
    CONSTRAINT "contracts_address_key" UNIQUE ("address")
);

-- CreateTable
CREATE TABLE IF NOT EXISTS "functions" (
    "id" BIGSERIAL NOT NULL,
    "name" VARCHAR(100) NOT NULL,
    "encodedName" VARCHAR(10) NOT NULL,
    "contract_id" BIGINT NOT NULL,
    CONSTRAINT "functions_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "functions_encodedName_contract_id_key" UNIQUE ("encodedName", "contract_id")
);

-- CreateTable
CREATE TABLE IF NOT EXISTS "transactions" (
    "transaction_id" BIGSERIAL NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL DEFAULT '2000-01-01 00:00:00 +00:00',
    "from" VARCHAR(42) NOT NULL,
    "to" VARCHAR(42) NOT NULL,
    "input" VARCHAR(1024) NOT NULL,
    "gas" VARCHAR(20) NOT NULL,
    "value" VARCHAR(20) NOT NULL,
    "chainId" VARCHAR(20) NOT NULL,
    "gasPrice" VARCHAR(20) NOT NULL,
    "nonce" VARCHAR(20) NOT NULL,
    "v" VARCHAR(66) NOT NULL,
    "r" VARCHAR(66) NOT NULL,
    "s" VARCHAR(66) NOT NULL,
    "rawTx" VARCHAR(1024) NOT NULL,
    "signedRawTx" VARCHAR(1024),
    "succeed" BOOLEAN,
    "function_id" BIGINT,
    "contract_id" BIGINT,
    "reporter_id" BIGINT,
    CONSTRAINT "transactions_pkey" PRIMARY KEY ("transaction_id")
);

-- CreateTable
CREATE TABLE IF NOT EXISTS "fee_payers" (
    "privateKey" VARCHAR(66) NOT NULL,
    CONSTRAINT "fee_payers_privateKey_key" UNIQUE ("privateKey")
);

-- CreateTable
CREATE TABLE IF NOT EXISTS "_ContractToReporter" (
    "A" BIGINT NOT NULL,
    "B" BIGINT NOT NULL,
    CONSTRAINT "_ContractToReporter_AB_unique" UNIQUE ("A", "B")
);

-- CreateIndex
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_class c
        JOIN   pg_namespace n ON n.oid = c.relnamespace
        WHERE  c.relname = '_ContractToReporter_B_index'
        AND    n.nspname = 'public'
    ) THEN
        CREATE INDEX "_ContractToReporter_B_index" ON "_ContractToReporter"("B");
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = 'reporters_organization_id_fkey'
    ) THEN
        ALTER TABLE "reporters" ADD CONSTRAINT "reporters_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("organization_id") ON DELETE CASCADE ON UPDATE CASCADE;
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = 'functions_contract_id_fkey'
    ) THEN
        ALTER TABLE "functions" ADD CONSTRAINT "functions_contract_id_fkey" FOREIGN KEY ("contract_id") REFERENCES "contracts"("contract_id") ON DELETE RESTRICT ON UPDATE CASCADE;
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = 'transactions_function_id_fkey'
    ) THEN
        ALTER TABLE "transactions" ADD CONSTRAINT "transactions_function_id_fkey" FOREIGN KEY ("function_id") REFERENCES "functions"("id") ON DELETE SET NULL ON UPDATE CASCADE;
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = 'transactions_contract_id_fkey'
    ) THEN
        ALTER TABLE "transactions" ADD CONSTRAINT "transactions_contract_id_fkey" FOREIGN KEY ("contract_id") REFERENCES "contracts"("contract_id") ON DELETE SET NULL ON UPDATE CASCADE;
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = 'transactions_reporter_id_fkey'
    ) THEN
        ALTER TABLE "transactions" ADD CONSTRAINT "transactions_reporter_id_fkey" FOREIGN KEY ("reporter_id") REFERENCES "reporters"("id") ON DELETE SET NULL ON UPDATE CASCADE;
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = '_ContractToReporter_A_fkey'
    ) THEN
        ALTER TABLE "_ContractToReporter" ADD CONSTRAINT "_ContractToReporter_A_fkey" FOREIGN KEY ("A") REFERENCES "contracts"("contract_id") ON DELETE CASCADE ON UPDATE CASCADE;
    END IF;
END $$;

-- AddForeignKey
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM   pg_constraint
        WHERE  conname = '_ContractToReporter_B_fkey'
    ) THEN
        ALTER TABLE "_ContractToReporter" ADD CONSTRAINT "_ContractToReporter_B_fkey" FOREIGN KEY ("B") REFERENCES "reporters"("id") ON DELETE CASCADE ON UPDATE CASCADE;
    END IF;
END $$;