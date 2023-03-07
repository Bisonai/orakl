-- CreateTable
CREATE TABLE "Transaction" (
    "id" SERIAL NOT NULL,
    "tx" TEXT NOT NULL,
    "signed" TEXT NOT NULL,

    CONSTRAINT "Transaction_pkey" PRIMARY KEY ("id")
);
