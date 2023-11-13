-- DropForeignKey
ALTER TABLE "last_submission" DROP CONSTRAINT "last_submission_aggregator_id_fkey";

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
ALTER TABLE "last_submission" ADD CONSTRAINT "last_submission_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "aggregators"("aggregator_id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "L2AggregatorPair" ADD CONSTRAINT "L2AggregatorPair_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "chains"("chain_id") ON DELETE RESTRICT ON UPDATE CASCADE;
