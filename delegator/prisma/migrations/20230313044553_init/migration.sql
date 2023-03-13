-- CreateTable
CREATE TABLE "Transaction" (
    "id" BIGSERIAL NOT NULL,
    "from" TEXT NOT NULL,
    "to" TEXT NOT NULL,
    "input" TEXT NOT NULL,
    "gas" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    "chainId" TEXT NOT NULL,
    "gasPrice" TEXT NOT NULL,
    "nonce" TEXT NOT NULL,
    "v" TEXT NOT NULL,
    "r" TEXT NOT NULL,
    "s" TEXT NOT NULL,
    "rawTx" TEXT NOT NULL,
    "signedRawTx" TEXT,

    CONSTRAINT "Transaction_pkey" PRIMARY KEY ("id")
);
