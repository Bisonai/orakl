-- CreateTable
CREATE TABLE "fee_payers" (
    "privateKey" VARCHAR(66) NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "fee_payers_privateKey_key" ON "fee_payers"("privateKey");
