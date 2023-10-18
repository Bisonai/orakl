-- CreateTable
CREATE TABLE "last_submission" (
    "submission_id" BIGSERIAL NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "value" BIGINT NOT NULL,
    "aggregator_id" BIGINT NOT NULL,

    CONSTRAINT "last_submission_pkey" PRIMARY KEY ("submission_id")
);

-- CreateIndex
CREATE UNIQUE INDEX "last_submission_aggregator_id_key" ON "last_submission"("aggregator_id");

-- AddForeignKey
ALTER TABLE "last_submission" ADD CONSTRAINT "last_submission_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "aggregators"("aggregator_id") ON DELETE CASCADE ON UPDATE CASCADE;
