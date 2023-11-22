/*
  Warnings:

  - A unique constraint covering the columns `[address,chain_id]` on the table `reporters` will be added. If there are existing duplicate values, this will fail.

*/
-- DropIndex
DROP INDEX "reporters_address_chain_id_service_id_key";

-- CreateIndex
CREATE UNIQUE INDEX "reporters_address_chain_id_key" ON "reporters"("address", "chain_id");
