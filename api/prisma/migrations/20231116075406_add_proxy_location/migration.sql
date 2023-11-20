/*
  Warnings:

  - Added the required column `location` to the `proxies` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "proxies" ADD COLUMN     "location" TEXT NOT NULL;