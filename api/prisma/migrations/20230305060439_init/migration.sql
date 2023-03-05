-- CreateTable
CREATE TABLE "Chain" (
    "id" SERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "Chain_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Feed" (
    "id" SERIAL NOT NULL,
    "name" TEXT NOT NULL,
    "latestRound" INTEGER NOT NULL,
    "definition" JSONB NOT NULL,
    "adapterId" INTEGER NOT NULL,

    CONSTRAINT "Feed_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Adapter" (
    "id" SERIAL NOT NULL,
    "adapterId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "decimals" INTEGER NOT NULL,

    CONSTRAINT "Adapter_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Aggregator" (
    "id" SERIAL NOT NULL,
    "aggregatorId" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT NOT NULL,
    "address" TEXT NOT NULL,
    "heartbeat" INTEGER NOT NULL,
    "threshold" DOUBLE PRECISION NOT NULL,
    "absoluteThreshold" DOUBLE PRECISION NOT NULL,
    "adapterId" INTEGER NOT NULL,
    "chainId" INTEGER NOT NULL,

    CONSTRAINT "Aggregator_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Data" (
    "id" BIGSERIAL NOT NULL,
    "round" BIGINT NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "value" INTEGER NOT NULL,
    "aggregatorId" INTEGER NOT NULL,
    "feedId" INTEGER NOT NULL,

    CONSTRAINT "Data_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "Chain_name_key" ON "Chain"("name");

-- CreateIndex
CREATE UNIQUE INDEX "Adapter_adapterId_key" ON "Adapter"("adapterId");

-- CreateIndex
CREATE UNIQUE INDEX "Aggregator_address_key" ON "Aggregator"("address");

-- CreateIndex
CREATE UNIQUE INDEX "Aggregator_aggregatorId_chainId_key" ON "Aggregator"("aggregatorId", "chainId");

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
