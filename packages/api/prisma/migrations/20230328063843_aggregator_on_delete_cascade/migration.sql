-- DropForeignKey
ALTER TABLE "aggregates" DROP CONSTRAINT "aggregates_aggregator_id_fkey";

-- DropForeignKey
ALTER TABLE "data" DROP CONSTRAINT "data_aggregator_id_fkey";

-- AddForeignKey
ALTER TABLE "data" ADD CONSTRAINT "data_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "aggregators"("aggregator_id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "aggregates" ADD CONSTRAINT "aggregates_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "aggregators"("aggregator_id") ON DELETE CASCADE ON UPDATE CASCADE;
