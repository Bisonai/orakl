-- CreateTable
CREATE TABLE "reporters" (
    "reporter_id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "privateKey" VARCHAR(66) NOT NULL,
    "oracleAddress" VARCHAR(42) NOT NULL,
    "chain_id" BIGINT NOT NULL,
    "service_id" BIGINT NOT NULL,

    CONSTRAINT "reporters_pkey" PRIMARY KEY ("reporter_id")
);

-- AddForeignKey
ALTER TABLE "reporters" ADD CONSTRAINT "reporters_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reporters" ADD CONSTRAINT "reporters_service_id_fkey" FOREIGN KEY ("service_id") REFERENCES "services"("service_id") ON DELETE RESTRICT ON UPDATE CASCADE;
