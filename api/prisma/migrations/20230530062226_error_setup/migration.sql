-- CreateTable
CREATE TABLE "error" (
    "error_id" BIGSERIAL NOT NULL,
    "request_id" VARCHAR(77) NOT NULL,
    "timestamp" TEXT NOT NULL,
    "errorMsg" TEXT NOT NULL,

    CONSTRAINT "error_pkey" PRIMARY KEY ("error_id")
);
