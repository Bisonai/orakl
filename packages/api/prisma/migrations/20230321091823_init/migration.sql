/*
  Warnings:

  - You are about to drop the `Chain` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "aggregators" DROP CONSTRAINT "aggregators_chain_id_fkey";

-- DropForeignKey
ALTER TABLE "listeners" DROP CONSTRAINT "listeners_chain_id_fkey";

-- DropForeignKey
ALTER TABLE "vrf_keys" DROP CONSTRAINT "vrf_keys_chain_id_fkey";

-- DropTable
DROP TABLE "Chain";

-- CreateTable
CREATE TABLE "chains" (
    "chain_id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "chains_pkey" PRIMARY KEY ("chain_id")
);

-- CreateIndex
CREATE UNIQUE INDEX "chains_name_key" ON "chains"("name");

-- AddForeignKey
ALTER TABLE "listeners" ADD CONSTRAINT "listeners_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vrf_keys" ADD CONSTRAINT "vrf_keys_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "aggregators" ADD CONSTRAINT "aggregators_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;
