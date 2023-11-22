-- CreateTable
CREATE TABLE "proxies" (
    "id" BIGSERIAL NOT NULL,
    "protocol" TEXT NOT NULL,
    "host" TEXT NOT NULL,
    "port" TEXT NOT NULL,

    CONSTRAINT "proxies_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "proxies_protocol_host_port_key" ON "proxies"("protocol", "host", "port");
