/*
  Warnings:

  - The primary key for the `Chain` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - You are about to drop the column `id` on the `Chain` table. All the data in the column will be lost.
  - You are about to drop the `Adapter` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Aggregate` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Aggregator` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Data` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Feed` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Listener` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `Service` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `VrfKey` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "Aggregate" DROP CONSTRAINT "Aggregate_aggregatorId_fkey";

-- DropForeignKey
ALTER TABLE "Aggregator" DROP CONSTRAINT "Aggregator_adapterId_fkey";

-- DropForeignKey
ALTER TABLE "Aggregator" DROP CONSTRAINT "Aggregator_chainId_fkey";

-- DropForeignKey
ALTER TABLE "Data" DROP CONSTRAINT "Data_aggregatorId_fkey";

-- DropForeignKey
ALTER TABLE "Data" DROP CONSTRAINT "Data_feedId_fkey";

-- DropForeignKey
ALTER TABLE "Feed" DROP CONSTRAINT "Feed_adapterId_fkey";

-- DropForeignKey
ALTER TABLE "Listener" DROP CONSTRAINT "Listener_chainId_fkey";

-- DropForeignKey
ALTER TABLE "Listener" DROP CONSTRAINT "Listener_serviceId_fkey";

-- DropForeignKey
ALTER TABLE "VrfKey" DROP CONSTRAINT "VrfKey_chainId_fkey";

-- AlterTable
ALTER TABLE "Chain" DROP CONSTRAINT "Chain_pkey",
DROP COLUMN "id",
ADD COLUMN     "chain_id" BIGSERIAL NOT NULL,
ADD CONSTRAINT "Chain_pkey" PRIMARY KEY ("chain_id");

-- DropTable
DROP TABLE "Adapter";

-- DropTable
DROP TABLE "Aggregate";

-- DropTable
DROP TABLE "Aggregator";

-- DropTable
DROP TABLE "Data";

-- DropTable
DROP TABLE "Feed";

-- DropTable
DROP TABLE "Listener";

-- DropTable
DROP TABLE "Service";

-- DropTable
DROP TABLE "VrfKey";

-- CreateTable
CREATE TABLE "services" (
    "service_id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "services_pkey" PRIMARY KEY ("service_id")
);

-- CreateTable
CREATE TABLE "listeners" (
    "listener_id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "event_name" VARCHAR(255) NOT NULL,
    "chain_id" BIGINT NOT NULL,
    "service_id" BIGINT NOT NULL,

    CONSTRAINT "listeners_pkey" PRIMARY KEY ("listener_id")
);

-- CreateTable
CREATE TABLE "vrf_keys" (
    "vrf_key_id" BIGSERIAL NOT NULL,
    "sk" VARCHAR(64) NOT NULL,
    "pk" VARCHAR(130) NOT NULL,
    "pk_x" VARCHAR(77) NOT NULL,
    "pk_y" VARCHAR(77) NOT NULL,
    "key_hash" VARCHAR(66) NOT NULL,
    "chain_id" BIGINT NOT NULL,

    CONSTRAINT "vrf_keys_pkey" PRIMARY KEY ("vrf_key_id")
);

-- CreateTable
CREATE TABLE "feeds" (
    "feed_id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,
    "definition" JSONB NOT NULL,
    "adapter_id" BIGINT NOT NULL,

    CONSTRAINT "feeds_pkey" PRIMARY KEY ("feed_id")
);

-- CreateTable
CREATE TABLE "adapters" (
    "adapter_id" BIGSERIAL NOT NULL,
    "adapter_hash" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "decimals" INTEGER NOT NULL,

    CONSTRAINT "adapters_pkey" PRIMARY KEY ("adapter_id")
);

-- CreateTable
CREATE TABLE "aggregators" (
    "aggregator_id" BIGSERIAL NOT NULL,
    "aggregator_hash" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT NOT NULL,
    "address" TEXT NOT NULL,
    "heartbeat" INTEGER NOT NULL,
    "threshold" DOUBLE PRECISION NOT NULL,
    "absolute_threshold" DOUBLE PRECISION NOT NULL,
    "adapter_id" BIGINT NOT NULL,
    "chain_id" BIGINT NOT NULL,

    CONSTRAINT "aggregators_pkey" PRIMARY KEY ("aggregator_id")
);

-- CreateTable
CREATE TABLE "data" (
    "data_id" BIGSERIAL NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "value" BIGINT NOT NULL,
    "aggregator_id" BIGINT NOT NULL,
    "feed_id" BIGINT NOT NULL,

    CONSTRAINT "data_pkey" PRIMARY KEY ("data_id")
);

-- CreateTable
CREATE TABLE "aggregates" (
    "aggregate_id" BIGSERIAL NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "value" BIGINT NOT NULL,
    "aggregator_id" BIGINT NOT NULL,

    CONSTRAINT "aggregates_pkey" PRIMARY KEY ("aggregate_id")
);

-- CreateIndex
CREATE UNIQUE INDEX "services_name_key" ON "services"("name");

-- CreateIndex
CREATE UNIQUE INDEX "adapters_adapter_hash_key" ON "adapters"("adapter_hash");

-- CreateIndex
CREATE UNIQUE INDEX "aggregators_address_key" ON "aggregators"("address");

-- CreateIndex
CREATE UNIQUE INDEX "aggregators_aggregator_hash_chain_id_key" ON "aggregators"("aggregator_hash", "chain_id");

-- CreateIndex
CREATE INDEX "aggregates_aggregator_id_timestamp_idx" ON "aggregates"("aggregator_id", "timestamp" DESC);

-- AddForeignKey
ALTER TABLE "listeners" ADD CONSTRAINT "listeners_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "Chain"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "listeners" ADD CONSTRAINT "listeners_service_id_fkey" FOREIGN KEY ("service_id") REFERENCES "services"("service_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vrf_keys" ADD CONSTRAINT "vrf_keys_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "Chain"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "feeds" ADD CONSTRAINT "feeds_adapter_id_fkey" FOREIGN KEY ("adapter_id") REFERENCES "adapters"("adapter_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "aggregators" ADD CONSTRAINT "aggregators_adapter_id_fkey" FOREIGN KEY ("adapter_id") REFERENCES "adapters"("adapter_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "aggregators" ADD CONSTRAINT "aggregators_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "Chain"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "data" ADD CONSTRAINT "data_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "aggregators"("aggregator_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "data" ADD CONSTRAINT "data_feed_id_fkey" FOREIGN KEY ("feed_id") REFERENCES "feeds"("feed_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "aggregates" ADD CONSTRAINT "aggregates_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "aggregators"("aggregator_id") ON DELETE RESTRICT ON UPDATE CASCADE;
