/*
  Warnings:

  - Changed the type of `port` on the `proxies` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- AlterTable
ALTER TABLE "proxies" DROP COLUMN "port",
ADD COLUMN     "port" INTEGER NOT NULL;

-- CreateIndex
CREATE UNIQUE INDEX "proxies_protocol_host_port_key" ON "proxies"("protocol", "host", "port");
