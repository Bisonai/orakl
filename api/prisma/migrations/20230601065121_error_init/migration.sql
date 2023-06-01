-- CreateTable
CREATE TABLE "error" (
    "error_id" BIGSERIAL NOT NULL,
    "request_id" TEXT NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    "code" BIGINT NOT NULL,
    "name" TEXT NOT NULL,
    "stack" TEXT NOT NULL,

    CONSTRAINT "error_pkey" PRIMARY KEY ("error_id")
);
