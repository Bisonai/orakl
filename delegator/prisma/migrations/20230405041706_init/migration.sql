-- CreateTable
CREATE TABLE "organizations" (
    "organization_id" BIGSERIAL NOT NULL,
    "name" VARCHAR(50) NOT NULL,

    CONSTRAINT "organizations_pkey" PRIMARY KEY ("organization_id")
);

-- CreateTable
CREATE TABLE "reporters" (
    "id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "organization_id" BIGINT NOT NULL,

    CONSTRAINT "reporters_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "contracts" (
    "contract_id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "allowAllFunctions" BOOLEAN DEFAULT false,

    CONSTRAINT "contracts_pkey" PRIMARY KEY ("contract_id")
);

-- CreateTable
CREATE TABLE "functions" (
    "id" BIGSERIAL NOT NULL,
    "name" VARCHAR(100) NOT NULL,
    "encodedName" VARCHAR(10) NOT NULL,
    "contract_id" BIGINT NOT NULL,

    CONSTRAINT "functions_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "transactions" (
    "transaction_id" BIGSERIAL NOT NULL,
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
    "reporterId" BIGINT,

    CONSTRAINT "transactions_pkey" PRIMARY KEY ("transaction_id")
);

-- CreateTable
CREATE TABLE "_WhitelistTable" (
    "A" BIGINT NOT NULL,
    "B" BIGINT NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "organizations_name_key" ON "organizations"("name");

-- CreateIndex
CREATE UNIQUE INDEX "reporters_address_key" ON "reporters"("address");

-- CreateIndex
CREATE UNIQUE INDEX "contracts_address_key" ON "contracts"("address");

-- CreateIndex
CREATE UNIQUE INDEX "functions_encodedName_key" ON "functions"("encodedName");

-- CreateIndex
CREATE UNIQUE INDEX "_WhitelistTable_AB_unique" ON "_WhitelistTable"("A", "B");

-- CreateIndex
CREATE INDEX "_WhitelistTable_B_index" ON "_WhitelistTable"("B");

-- AddForeignKey
ALTER TABLE "reporters" ADD CONSTRAINT "reporters_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations"("organization_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "functions" ADD CONSTRAINT "functions_contract_id_fkey" FOREIGN KEY ("contract_id") REFERENCES "contracts"("contract_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transactions" ADD CONSTRAINT "transactions_function_id_fkey" FOREIGN KEY ("function_id") REFERENCES "functions"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transactions" ADD CONSTRAINT "transactions_contract_id_fkey" FOREIGN KEY ("contract_id") REFERENCES "contracts"("contract_id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transactions" ADD CONSTRAINT "transactions_reporterId_fkey" FOREIGN KEY ("reporterId") REFERENCES "reporters"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_WhitelistTable" ADD CONSTRAINT "_WhitelistTable_A_fkey" FOREIGN KEY ("A") REFERENCES "contracts"("contract_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_WhitelistTable" ADD CONSTRAINT "_WhitelistTable_B_fkey" FOREIGN KEY ("B") REFERENCES "reporters"("id") ON DELETE CASCADE ON UPDATE CASCADE;
