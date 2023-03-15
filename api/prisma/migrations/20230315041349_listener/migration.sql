-- CreateTable
CREATE TABLE "Listener" (
    "id" BIGSERIAL NOT NULL,
    "address" VARCHAR(42) NOT NULL,
    "eventName" VARCHAR(255) NOT NULL,
    "chainId" BIGINT NOT NULL,
    "serviceId" BIGINT NOT NULL,

    CONSTRAINT "Listener_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "Listener" ADD CONSTRAINT "Listener_chainId_fkey" FOREIGN KEY ("chainId") REFERENCES "Chain"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Listener" ADD CONSTRAINT "Listener_serviceId_fkey" FOREIGN KEY ("serviceId") REFERENCES "Service"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
