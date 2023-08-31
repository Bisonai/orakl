/*
  Warnings:

  - You are about to alter the column `fetcher_type` on the `aggregators` table. The data in that column could be lost. The data in that column will be cast from `BigInt` to `Integer`.

*/
-- AlterTable
ALTER TABLE "aggregators" ALTER COLUMN "fetcher_type" SET DEFAULT 0,
ALTER COLUMN "fetcher_type" SET DATA TYPE INTEGER;
