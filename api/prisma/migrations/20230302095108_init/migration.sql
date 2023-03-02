-- CreateTable
CREATE TABLE "Feed" (
    "id" SERIAL NOT NULL,
    "source" TEXT NOT NULL,
    "decimals" INTEGER NOT NULL,
    "latestRound" INTEGER NOT NULL,
    "adapterId" INTEGER NOT NULL,

    CONSTRAINT "Feed_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Adapter" (
    "id" SERIAL NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "Adapter_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "Feed" ADD CONSTRAINT "Feed_adapterId_fkey" FOREIGN KEY ("adapterId") REFERENCES "Adapter"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
