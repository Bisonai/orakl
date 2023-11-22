-- CreateTable
CREATE TABLE "Chain" (
    "id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "Chain_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Service" (
    "id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "Service_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Feed" (
    "id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,
    "definition" JSONB NOT NULL,
    "adapterId" BIGINT NOT NULL,

    CONSTRAINT "Feed_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Adapter" (
    "id" BIGSERIAL NOT NULL,
    "adapterHash" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "decimals" INTEGER NOT NULL,

    CONSTRAINT "Adapter_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Aggregator" (
    "id" BIGSERIAL NOT NULL,
    "aggregatorHash" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT NOT NULL,
    "address" TEXT NOT NULL,
    "heartbeat" INTEGER NOT NULL,
    "threshold" DOUBLE PRECISION NOT NULL,
    "absoluteThreshold" DOUBLE PRECISION NOT NULL,
    "adapterId" BIGINT NOT NULL,
    "chainId" BIGINT NOT NULL,

    CONSTRAINT "Aggregator_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Data" (
    "id" BIGSERIAL NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "value" BIGINT NOT NULL,
    "aggregatorId" BIGINT NOT NULL,
    "feedId" BIGINT NOT NULL,

    CONSTRAINT "Data_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Aggregate" (
    "id" BIGSERIAL NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "value" BIGINT NOT NULL,
    "aggregatorId" BIGINT NOT NULL,

    CONSTRAINT "Aggregate_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "Chain_name_key" ON "Chain"("name");

-- CreateIndex
CREATE UNIQUE INDEX "Service_name_key" ON "Service"("name");

-- CreateIndex
CREATE UNIQUE INDEX "Adapter_adapterHash_key" ON "Adapter"("adapterHash");

-- CreateIndex
CREATE UNIQUE INDEX "Aggregator_address_key" ON "Aggregator"("address");

-- CreateIndex
CREATE UNIQUE INDEX "Aggregator_aggregatorHash_chainId_key" ON "Aggregator"("aggregatorHash", "chainId");

-- CreateIndex
CREATE INDEX "Aggregate_aggregatorId_timestamp_idx" ON "Aggregate"("aggregatorId", "timestamp" DESC);

-- AddForeignKey
ALTER TABLE "Feed" ADD CONSTRAINT "Feed_adapterId_fkey" FOREIGN KEY ("adapterId") REFERENCES "Adapter"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Aggregator" ADD CONSTRAINT "Aggregator_adapterId_fkey" FOREIGN KEY ("adapterId") REFERENCES "Adapter"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Aggregator" ADD CONSTRAINT "Aggregator_chainId_fkey" FOREIGN KEY ("chainId") REFERENCES "Chain"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Data" ADD CONSTRAINT "Data_aggregatorId_fkey" FOREIGN KEY ("aggregatorId") REFERENCES "Aggregator"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Data" ADD CONSTRAINT "Data_feedId_fkey" FOREIGN KEY ("feedId") REFERENCES "Feed"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Aggregate" ADD CONSTRAINT "Aggregate_aggregatorId_fkey" FOREIGN KEY ("aggregatorId") REFERENCES "Aggregator"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
