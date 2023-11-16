/*
  Warnings:

  - Added the required column `location` to the `proxies` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "proxies" ADD COLUMN     "location" TEXT NOT NULL;

-- CreateTable
CREATE TABLE "L2AggregatorPair" (
    "id" BIGSERIAL NOT NULL,
    "l1_aggregator_addresss" TEXT NOT NULL,
    "l2_aggregator_addresss" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "chain_id" BIGINT NOT NULL,

    CONSTRAINT "L2AggregatorPair_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "L2AggregatorPair" ADD CONSTRAINT "L2AggregatorPair_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;
