-- CreateTable
CREATE TABLE "chains" (
    "chain_id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "chains_pkey" PRIMARY KEY ("chain_id")
);

-- CreateTable
CREATE TABLE "services" (
    "service_id" BIGSERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "services_pkey" PRIMARY KEY ("service_id")
);

-- CreateTable
CREATE TABLE "l1_aggregators" (
    "l1_aggregator_id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "event_name" VARCHAR(255) NOT NULL,
    "chain_id" BIGINT NOT NULL,
    "service_id" BIGINT NOT NULL,

    CONSTRAINT "l1_aggregators_pkey" PRIMARY KEY ("l1_aggregator_id")
);

-- CreateTable
CREATE TABLE "l2_endpoints" (
    "l2_endpoint_id" BIGSERIAL NOT NULL,
    "owner" VARCHAR(42) NOT NULL,
    "jsonRpc" VARCHAR(255) NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "l1_aggregator_id" BIGINT NOT NULL,
    "l2Aggregator" VARCHAR(42) NOT NULL,

    CONSTRAINT "l2_endpoints_pkey" PRIMARY KEY ("l2_endpoint_id")
);

-- CreateTable
CREATE TABLE "reporters" (
    "reporter_id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "privateKey" VARCHAR(164) NOT NULL,
    "chain_id" BIGINT NOT NULL,
    "service_id" BIGINT NOT NULL,
    "l2_endpoint_id" BIGINT NOT NULL,

    CONSTRAINT "reporters_pkey" PRIMARY KEY ("reporter_id")
);

-- CreateTable
CREATE TABLE "reports" (
    "report_id" BIGSERIAL NOT NULL,
    "value" BIGINT NOT NULL,
    "roundId" BIGINT NOT NULL,
    "reporter_id" BIGINT NOT NULL,

    CONSTRAINT "reports_pkey" PRIMARY KEY ("report_id")
);

-- CreateIndex
CREATE UNIQUE INDEX "chains_name_key" ON "chains"("name");

-- CreateIndex
CREATE UNIQUE INDEX "services_name_key" ON "services"("name");

-- CreateIndex
CREATE UNIQUE INDEX "l1_aggregators_address_key" ON "l1_aggregators"("address");

-- CreateIndex
CREATE UNIQUE INDEX "l2_endpoints_owner_key" ON "l2_endpoints"("owner");

-- CreateIndex
CREATE UNIQUE INDEX "l2_endpoints_l2Aggregator_key" ON "l2_endpoints"("l2Aggregator");

-- AddForeignKey
ALTER TABLE "l1_aggregators" ADD CONSTRAINT "l1_aggregators_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "l1_aggregators" ADD CONSTRAINT "l1_aggregators_service_id_fkey" FOREIGN KEY ("service_id") REFERENCES "services"("service_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "l2_endpoints" ADD CONSTRAINT "l2_endpoints_l1_aggregator_id_fkey" FOREIGN KEY ("l1_aggregator_id") REFERENCES "l1_aggregators"("l1_aggregator_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reporters" ADD CONSTRAINT "reporters_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reporters" ADD CONSTRAINT "reporters_service_id_fkey" FOREIGN KEY ("service_id") REFERENCES "services"("service_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reporters" ADD CONSTRAINT "reporters_l2_endpoint_id_fkey" FOREIGN KEY ("l2_endpoint_id") REFERENCES "l2_endpoints"("l2_endpoint_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reports" ADD CONSTRAINT "reports_reporter_id_fkey" FOREIGN KEY ("reporter_id") REFERENCES "reporters"("reporter_id") ON DELETE RESTRICT ON UPDATE CASCADE;
